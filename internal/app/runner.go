package app

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"
)

var (
	ErrBusy   = newError("runner_busy", "Another task is running. Try again shortly.")
	ErrClosed = newError("service_stopping", "The service is shutting down.")
)

type Runner struct {
	db        *gorm.DB
	microsoft *Microsoft
	ctx       context.Context
	cancel    context.CancelFunc
	scheduler gocron.Scheduler
	mu        sync.Mutex
	busy      bool
	closed    bool
	wg        sync.WaitGroup
	random    *rand.Rand
}

// NewRunner owns scheduling and the single background execution slot.
func NewRunner(db *gorm.DB, microsoft *Microsoft) (*Runner, error) {
	ctx, cancel := context.WithCancel(context.Background())
	scheduler, err := gocron.NewScheduler(
		gocron.WithLogger(slog.Default()),
		gocron.WithGlobalJobOptions(
			gocron.WithEventListeners(gocron.AfterJobRunsWithPanic(func(id uuid.UUID, name string, recovered any) {
				logPanic(
					ctx,
					"scheduler job panicked",
					recovered,
					slog.String("job_id", id.String()),
					slog.String("job_name", name),
				)
			})),
		),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	r := &Runner{
		db:        db,
		microsoft: microsoft,
		ctx:       ctx,
		cancel:    cancel,
		scheduler: scheduler,
		random:    rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}

	// Persisted due times let one poll recover overdue work after a restart.
	_, err = scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(r.Tick),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		cancel()
		return nil, errors.Join(err, scheduler.Shutdown())
	}

	if _, err := scheduler.NewJob(
		gocron.DurationJob(24*time.Hour),
		gocron.NewTask(r.prune),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		cancel()
		return nil, errors.Join(err, scheduler.Shutdown())
	}

	return r, nil
}

// Start marks interrupted work before scheduling new executions.
func (r *Runner) Start() error {
	now := time.Now().UTC()
	if err := r.db.Model(&Run{}).Where("status = ?", RunRunning).Updates(map[string]any{
		"status":      RunInterrupted,
		"error":       "The service stopped during the previous execution.",
		"error_code":  "execution_interrupted",
		"finished_at": now,
	}).Error; err != nil {
		return err
	}

	r.scheduler.Start()

	return nil
}

// Close stops polling, cancels active work, and waits for result persistence.
func (r *Runner) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}

	r.closed = true
	r.cancel()
	r.mu.Unlock()
	if err := r.scheduler.Shutdown(); err != nil {
		slog.Error("scheduler shutdown failed", "error", err)
	}

	r.wg.Wait()
}

// Busy reports whether an execution has reserved the worker slot.
func (r *Runner) Busy() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.busy
}

// Tick starts the earliest due account; other due accounts remain for later polls.
func (r *Runner) Tick() {
	if r.Busy() {
		return
	}

	var account Account
	err := r.db.WithContext(r.ctx).
		Where("enabled = ? AND next_run_at <= ?", true, time.Now().UTC()).
		Order("next_run_at, id").
		First(&account).
		Error
	if err == nil {
		if _, err := r.Trigger(
			account.ID,
			TriggerScheduled,
		); err != nil && !errors.Is(err, ErrBusy) &&
			!errors.Is(err, ErrClosed) {
			logDatabaseError(
				r.ctx,
				"scheduled task could not start",
				err,
				slog.Uint64("account_id", uint64(account.ID)),
			)
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) && r.ctx.Err() == nil {
		logDatabaseError(r.ctx, "schedule query failed", err)
	}
}

func nextRun(a Account, now time.Time) *time.Time {
	minutes := a.MinMinutes
	if a.MaxMinutes > minutes {
		minutes += rand.IntN(a.MaxMinutes - minutes + 1)
	}

	next := now.Add(time.Duration(minutes) * time.Minute).UTC()

	return &next
}

func sampleEndpoints(authType AuthType, random *rand.Rand) []Endpoint {
	pool := endpointsFor(authType)
	if len(pool) == 0 {
		return nil
	}

	count := 1 + random.IntN(len(pool))
	random.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	return pool[:count]
}

// Trigger reserves the single execution slot before returning to HTTP callers.
func (r *Runner) Trigger(id uint, trigger RunTrigger) (*Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, ErrClosed
	}

	if !trigger.Valid() {
		return nil, newError("trigger_invalid", "Invalid execution trigger.")
	}

	if r.busy {
		return nil, ErrBusy
	}

	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	var account Account
	if err := r.db.WithContext(ctx).First(&account, id).Error; err != nil {
		return nil, err
	}

	if trigger == TriggerScheduled &&
		(!account.Enabled || account.NextRunAt == nil || account.NextRunAt.After(time.Now())) {
		return nil, nil
	}

	run := &Run{
		AccountID:   account.ID,
		AccountName: account.Name,
		Trigger:     trigger,
		Status:      RunRunning,
		StartedAt:   time.Now().UTC(),
		Results:     []GraphCall{},
	}
	if err := r.db.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}

	r.busy = true
	r.wg.Add(1)
	runCopy := *run
	go func() {
		defer r.wg.Done()
		defer func() {
			r.mu.Lock()
			r.busy = false
			r.mu.Unlock()
		}()
		// The entry boundary also covers failures in panic recovery and result persistence.
		defer func() {
			if recovered := recover(); recovered != nil {
				logPanic(r.ctx, "worker recovery failed", recovered, slog.Uint64("run_id", uint64(runCopy.ID)))
			}
		}()
		r.execute(account, runCopy)
	}()

	return run, nil
}

func (r *Runner) execute(a Account, run Run) {
	ctx, cancel := context.WithTimeout(r.ctx, 3*time.Minute)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logPanic(
				ctx,
				"execution panicked",
				recovered,
				slog.Uint64("run_id", uint64(run.ID)),
				slog.Uint64("account_id", uint64(a.ID)),
			)
			run.Status = RunFailed
			run.Error = "An unexpected error interrupted execution."
			run.ErrorCode = "execution_panicked"
			r.finish(a, run, false)
		}
	}()
	endpoints := sampleEndpoints(a.AuthType, r.random)
	if len(endpoints) == 0 {
		run.Status = RunFailed
		run.Error = "No Graph endpoints support this authorization type."
		run.ErrorCode = "auth_type_invalid"
		r.finish(a, run, false)

		return
	}

	token, err := r.microsoft.Token(ctx, a)

	var authErr *authFailure
	needsAuthorization := errors.As(err, &authErr)
	if err == nil {
		err = r.saveRefreshToken(ctx, a, token.RefreshToken)
	}

	if err == nil {
		err = r.callEndpoints(ctx, token.AccessToken, endpoints, &run)
	}

	run.Status = RunSuccess
	if err != nil {
		run.Status, run.Error = RunFailed, err.Error()
		run.ErrorCode = errorCode(err)
	} else {
		succeeded, failed := 0, 0
		for _, result := range run.Results {
			if result.Error != "" {
				failed++
			} else {
				succeeded++
			}
		}

		if failed > 0 {
			run.Status = RunFailed
			run.Error = "One or more Graph requests failed."
			run.ErrorCode = "graph_requests_failed"
			if succeeded > 0 {
				run.Status = RunPartial
				run.Error = "Some Graph requests succeeded and others failed."
				run.ErrorCode = "graph_requests_partial"
			}
		}
	}

	r.finish(a, run, needsAuthorization)
}

func (r *Runner) saveRefreshToken(ctx context.Context, account Account, refreshToken string) error {
	if refreshToken == "" || refreshToken == account.RefreshToken {
		return nil
	}

	// A refreshed token must not overwrite credentials edited during execution.
	result := r.db.WithContext(ctx).
		Model(&Account{}).
		Where(
			"id = ? AND revision = ? AND client_id = ? AND client_secret = ? AND refresh_token = ?",
			account.ID,
			account.Revision,
			account.ClientID,
			account.ClientSecret,
			account.RefreshToken,
		).
		Update("refresh_token", refreshToken)
	if result.Error != nil {
		logDatabaseError(
			ctx,
			"token rotation persistence failed",
			result.Error,
			slog.Uint64("account_id", uint64(account.ID)),
		)

		return newError("token_save_failed", "Could not persist the rotated refresh token.")
	}

	if result.RowsAffected != 1 {
		return newError("credentials_changed", "Account credentials changed during execution. Run the task again.")
	}

	return nil
}

func (r *Runner) callEndpoints(ctx context.Context, accessToken string, endpoints []Endpoint, run *Run) error {
	for _, endpoint := range endpoints {
		result := r.microsoft.Call(ctx, accessToken, endpoint)
		result.RunID = run.ID

		// Keep completed calls even if a later call or the process is interrupted.
		saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
		saveError := r.db.WithContext(saveCtx).Create(&result).Error
		saveCancel()
		if saveError != nil {
			logDatabaseError(ctx, "Graph result persistence failed", saveError, slog.Uint64("run_id", uint64(run.ID)))
			return newError("graph_call_save_failed", "Could not persist the Graph call result.")
		}

		run.Results = append(run.Results, result)
		if ctx.Err() != nil {
			if len(run.Results) < len(endpoints) {
				return newError("task_cancelled", "The task was cancelled or timed out.")
			}

			break
		}
	}

	return nil
}

func (r *Runner) finish(a Account, run Run, needsAuthorization bool) {
	finished := time.Now().UTC()
	run.FinishedAt = &finished
	// Persist cancellation results even after the application context is cancelled.
	saveCtx, saveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer saveCancel()
	err := r.db.WithContext(saveCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Results").Save(&run).Error; err != nil {
			return err
		}

		updates := map[string]any{"last_run_at": finished, "last_error": run.Error, "last_error_code": run.ErrorCode}
		if a.Enabled {
			updates["next_run_at"] = nextRun(a, finished)
		}

		if needsAuthorization {
			updates["enabled"], updates["next_run_at"] = false, nil
		}

		// Do not undo a concurrent schedule change or reauthorization.
		return tx.Model(&Account{}).Where("id = ? AND revision = ?", a.ID, a.Revision).Updates(updates).Error
	})
	if err != nil {
		logDatabaseError(saveCtx, "task result persistence failed", err, slog.Uint64("run_id", uint64(run.ID)))
		return
	}

	slog.Info(
		"execution completed",
		"run_id",
		run.ID,
		"account_id",
		a.ID,
		"status",
		run.Status,
		"error_code",
		run.ErrorCode,
	)
}

// Change serializes account edits with execution so token rotation cannot be lost.
func (r *Runner) Change(ctx context.Context, change func(*gorm.DB) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return ErrClosed
	}

	if r.busy {
		return ErrBusy
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.db.WithContext(ctx).Transaction(change)
}

func (r *Runner) prune() {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().AddDate(0, 0, -30)
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		oldRuns := tx.Model(&Run{}).Select("id").Where("started_at < ?", cutoff)
		if err := tx.Where("run_id IN (?)", oldRuns).Delete(&GraphCall{}).Error; err != nil {
			return err
		}

		return tx.Where("started_at < ?", cutoff).Delete(&Run{}).Error
	}); err != nil {
		logDatabaseError(ctx, "history retention failed", err)
	}
}
