package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/executor"
	"e5-renewal/backend/services/notifier"
)

// Scheduler manages periodic Graph API calls for all accounts
// and daily auth expiry checks.
type Scheduler struct {
	Executor *executor.Executor
	Notifier *notifier.Service
	Rand     *rand.Rand
	Now      func() time.Time // injectable clock

	mu     sync.Mutex
	wg     sync.WaitGroup
	timers map[uint]*time.Timer
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a scheduler instance.
func New(exec *executor.Executor, rng *rand.Rand) *Scheduler {
	return &Scheduler{
		Executor: exec,
		Notifier: notifier.NewService(),
		Rand:     rng,
		Now:      time.Now,
	}
}

func (s *Scheduler) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Start launches the scheduler goroutines.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, s.cancel = context.WithCancel(ctx)
	s.ctx = ctx
	s.timers = make(map[uint]*time.Timer)

	slog.Info("scheduler startup")

	schedules, err := database.Schedules.ListActive(ctx)
	if err != nil {
		slog.Error("failed to load active schedules", "subsystem", "scheduler", "error", err)
		schedules = nil
	}

	slog.Info("scheduler active schedules loaded", "count", len(schedules))

	now := s.now()
	for _, sched := range schedules {
		s.scheduleAccount(ctx, sched, now)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.authExpiryLoop(ctx)
	}()

	slog.Info("scheduler started", "subsystem", "scheduler", "accounts_scheduled", len(schedules))
}

// Stop cancels all running timers and waits for in-flight callbacks.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Lock()
	for _, t := range s.timers {
		if t.Stop() {
			// Timer was stopped before it fired; its wg.Done will never run.
			s.wg.Done()
		}
	}
	s.timers = nil
	s.mu.Unlock()
	s.wg.Wait()
}

// RegisterAccount adds or updates a schedule for an account.
func (s *Scheduler) RegisterAccount(ctx context.Context, accountID uint) {
	sched, err := database.Schedules.GetByAccountID(ctx, accountID)
	if err != nil {
		return
	}
	if !sched.Enabled || sched.Paused {
		s.removeTimer(accountID)
		return
	}
	s.scheduleAccount(ctx, *sched, s.now())
}

// UnregisterAccount removes the timer for an account.
func (s *Scheduler) UnregisterAccount(accountID uint) {
	s.removeTimer(accountID)
}

// TriggerNow manually triggers a run and returns the result synchronously.
func (s *Scheduler) TriggerNow(ctx context.Context, accountID uint) (*executor.TriggerResult, error) {
	account, err := database.Accounts.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.Executor.RunManual(ctx, *account)
}

// ComputeNextRun returns the next run time from now (exported for use by handlers).
func (s *Scheduler) ComputeNextRun() time.Time {
	return s.computeNextRun(s.now())
}

func (s *Scheduler) scheduleAccount(ctx context.Context, sched models.Schedule, now time.Time) {
	delay, nextRunAt, runImmediately, reason := s.scheduleDecision(sched, now)
	if nextRunAt != nil {
		if err := database.Schedules.UpdateFields(ctx, sched.AccountID, map[string]any{"next_run_at": *nextRunAt}); err != nil {
			slog.Error("failed to update next run", "subsystem", "scheduler", "account_id", sched.AccountID, "error", err)
		} else {
			sched.NextRunAt = nextRunAt
		}
	}

	slog.Info(
		"scheduler next-run decision",
		"account_id", sched.AccountID,
		"reason", reason,
		"delay", delay.String(),
		"run_immediately", runImmediately,
		"has_next_run_at", sched.NextRunAt != nil,
	)

	s.mu.Lock()
	if s.timers == nil {
		s.timers = make(map[uint]*time.Timer)
	}
	if old, ok := s.timers[sched.AccountID]; ok {
		if old.Stop() {
			// Old timer stopped before firing; its wg.Done will never run.
			s.wg.Done()
		}
	}
	s.wg.Add(1)
	timer := time.AfterFunc(delay, func() {
		defer s.wg.Done()
		s.runTask(ctx, sched.AccountID)
	})
	s.timers[sched.AccountID] = timer
	s.mu.Unlock()
}

func (s *Scheduler) scheduleDecision(sched models.Schedule, now time.Time) (time.Duration, *time.Time, bool, string) {
	if sched.NextRunAt != nil {
		if sched.NextRunAt.After(now) {
			return sched.NextRunAt.Sub(now), nil, false, "persisted_future"
		}
		slog.Info(
			"scheduler overdue catch-up decision",
			"account_id", sched.AccountID,
			"stored_next_run_at", sched.NextRunAt,
			"now", now,
			"action", "run_immediately_once",
		)
		return 0, nil, true, "overdue_catch_up"
	}

	nextRun := s.computeNextRun(now)
	return nextRun.Sub(now), &nextRun, false, "computed_next_run"
}

func (s *Scheduler) runTask(ctx context.Context, accountID uint) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	account, err := database.Accounts.GetByID(ctx, accountID)
	if err != nil {
		slog.Error("account not found", "subsystem", "scheduler", "account_id", accountID, "error", err)
		return
	}

	sched, err := database.Schedules.GetByAccountID(ctx, accountID)
	if err != nil {
		slog.Error("schedule not found", "subsystem", "scheduler", "account_id", accountID, "error", err)
		return
	}

	if !sched.Enabled || sched.Paused {
		return
	}

	// Load notification config before task execution so it is available for all notifications.
	var cfg models.NotificationConfig
	s.loadNotificationConfig(ctx, &cfg)
	lang := cfg.Language
	if lang == "" {
		lang = "zh"
	}

	slog.Info("running task", "subsystem", "scheduler", "account_id", accountID, "account_name", account.Name)

	taskLog, err := s.Executor.Run(ctx, *account, models.TriggerScheduled)
	if err != nil {
		slog.Error("task error", "subsystem", "scheduler", "account_id", accountID, "error", err)
	}

	now := s.now()
	nextRun := s.computeNextRun(now)
	updates := map[string]any{
		"last_run_at": now,
		"next_run_at": nextRun,
	}

	if sched.PauseThreshold > 0 {
		health := s.computeHealth(ctx, accountID)
		if health >= 0 && health < float64(sched.PauseThreshold) {
			updates["paused"] = true
			updates["pause_reason"] = fmt.Sprintf("Health %.0f%% below threshold %d%%", health, sched.PauseThreshold)
			slog.Warn("account auto-paused", "subsystem", "scheduler", "account_id", accountID, "health", health, "threshold", sched.PauseThreshold)

			hlTitle, hlMsg := notifier.FormatHealthLow(lang, account.Name, health, sched.PauseThreshold)
			s.notifyIfEnabled(&cfg, "on_health_low", hlTitle, hlMsg)
		}
	}

	if err := database.Schedules.UpdateFields(ctx, accountID, updates); err != nil {
		slog.Error("failed to update schedule", "subsystem", "scheduler", "account_id", accountID, "error", err)
	}

	if taskLog != nil && taskLog.FailCount > 0 && taskLog.SuccessCount == 0 {
		title, msg := notifier.FormatTaskAllFailed(lang, account.Name, taskLog.FailCount)
		s.notifyIfEnabled(&cfg, "on_task_all_failed", title, msg)
	}

	if _, paused := updates["paused"]; !paused {
		sched.NextRunAt = &nextRun
		s.scheduleAccount(ctx, *sched, s.now())
	}
}

func (s *Scheduler) computeNextRun(now time.Time) time.Time {
	cfg := config.Get().Scheduler
	minH := cfg.MinHours
	maxH := cfg.MaxHours
	if minH <= 0 {
		minH = 2
	}
	if maxH < minH {
		maxH = minH
	}

	hours := float64(minH)
	if maxH > minH {
		hours += float64(s.Rand.Intn(maxH - minH + 1))
	}

	if cfg.RealisticTiming {
		hours *= timeWeight(now)
	}

	return now.Add(time.Duration(hours*3600) * time.Second)
}

func timeWeight(t time.Time) float64 {
	hour := t.Hour()
	weekday := t.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday

	if isWeekend {
		return 1.5
	}

	switch {
	case hour >= 9 && hour < 18:
		return 0.5
	case hour >= 18 && hour < 23:
		return 1.0
	default:
		return 2.0
	}
}

func (s *Scheduler) computeHealth(ctx context.Context, accountID uint) float64 {
	logs, err := database.TaskLogs.Last20ByAccount(ctx, accountID)
	if err != nil || len(logs) == 0 {
		return -1
	}
	var totalEp, successEp int
	for i := range logs {
		totalEp += logs[i].TotalEndpoints
		successEp += logs[i].SuccessCount
	}
	if totalEp == 0 {
		return -1
	}
	return float64(successEp) / float64(totalEp) * 100
}

func (s *Scheduler) authExpiryLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	s.checkAuthExpiry(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAuthExpiry(ctx)
		}
	}
}

func (s *Scheduler) checkAuthExpiry(ctx context.Context) {
	var cfg models.NotificationConfig
	s.loadNotificationConfig(ctx, &cfg)

	if !cfg.OnAuthExpiry {
		return
	}

	daysBefore := cfg.ExpiryDaysBefore
	if daysBefore <= 0 {
		daysBefore = 7
	}

	threshold := s.now().AddDate(0, 0, daysBefore)

	accounts, err := database.Accounts.FindExpiringBefore(ctx, threshold)
	if err != nil {
		slog.Error("failed to find expiring accounts", "subsystem", "scheduler", "error", err)
		return
	}

	lang := cfg.Language
	if lang == "" {
		lang = "zh"
	}

	for _, acc := range accounts {
		daysLeft := int(acc.AuthExpiresAt.Sub(s.now()).Hours() / 24)
		title, msg := notifier.FormatAuthExpiry(lang, acc.Name, daysLeft)
		s.notifyIfEnabled(&cfg, "on_auth_expiry", title, msg)
	}
}

func (s *Scheduler) loadNotificationConfig(ctx context.Context, cfg *models.NotificationConfig) {
	val, err := database.Settings.Get(ctx, models.SettingKeyNotification)
	if err != nil || val == "" {
		return
	}
	_ = json.Unmarshal([]byte(val), cfg)
}

func (s *Scheduler) notifyIfEnabled(cfg *models.NotificationConfig, eventKey, title, message string) {
	if s.Notifier == nil {
		return
	}

	if cfg.URL == "" {
		return
	}

	var enabled bool
	switch eventKey {
	case "on_auth_expiry":
		enabled = cfg.OnAuthExpiry
	case "on_task_all_failed":
		enabled = cfg.OnTaskAllFailed
	case "on_health_low":
		enabled = cfg.OnHealthLow
	}

	if !enabled {
		return
	}

	if err := s.Notifier.Send(cfg.URL, title, message); err != nil {
		slog.Error("notification send failed", "subsystem", "scheduler", "error", err)
	}
}

func (s *Scheduler) removeTimer(accountID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.timers[accountID]; ok {
		if t.Stop() {
			// Timer was stopped before it fired; its wg.Done will never run.
			s.wg.Done()
		}
		delete(s.timers, accountID)
	}
}
