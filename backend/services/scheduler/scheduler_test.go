package scheduler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/executor"
	"e5-renewal/backend/services/graph"
	"e5-renewal/backend/services/oauth"
	"e5-renewal/backend/services/scheduler"
)

func newRng() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

func initTestDB(t *testing.T) {
	t.Helper()
	// Use a unique shared-cache in-memory DB per test to avoid cross-test data leaks
	// while ensuring all GORM connections within a test share the same DB.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	err := database.Init(dsn)
	require.NoError(t, err)
	database.MustInitEncryption("test-encryption-key-minimum16chars")
}

func initTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")
	// Limit to 1 endpoint per run so CallEndpoints has no inter-call delay in tests.
	cfg := config.Get()
	cfg.Scheduler.EndpointsMin = 1
	cfg.Scheduler.EndpointsMax = 1
}

func makeExecutor(tokenServerURL, graphServerURL string) *executor.Executor {
	rng := newRng()
	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServerURL},
	})
	exec := executor.New(oauthSvc, rng)
	exec.Graph = &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: graphServerURL}},
		Rand:       newRng(),
	}
	return exec
}

func TestNew(t *testing.T) {
	rng := newRng()
	exec := executor.New(oauth.NewService(nil), rng)
	sched := scheduler.New(exec, newRng())

	assert.NotNil(t, sched)
	assert.Equal(t, exec, sched.Executor)
}

func TestComputeNextRun_DefaultConfig(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) // Tuesday 2pm
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	assert.True(t, next.After(fixedNow))
	// With seed 42, Intn(5)=0, so hours = minHours (2).
	// RealisticTiming defaults to false, so no weight applied.
	diff := next.Sub(fixedNow)
	assert.InDelta(t, 2*3600, diff.Seconds(), 1)
}

func TestComputeNextRun_WithRandomOffset(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	diff := next.Sub(fixedNow)
	// minHours=2, maxHours=6. With seed 42, Intn(5)=0, so hours=2.
	assert.InDelta(t, 2*3600, diff.Seconds(), 1)
}

func TestStartStop(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)

	// Should not panic
	sched.Stop()
}

func TestStartStop_MultipleStops(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	sched.Stop()
	// Second stop should not panic
	sched.Stop()
}

func TestRegisterAccount_EnabledSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Create account and schedule
	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{Name: "reg-test", AuthType: models.AuthTypeAuthCode, AuthInfo: string(authJSON)}
	err := database.Accounts.Create(ctx, &account)
	require.NoError(t, err)

	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	err = database.Schedules.Create(ctx, &schedule)
	require.NoError(t, err)

	// Should not panic
	sched.RegisterAccount(ctx, account.ID)
}

func TestRegisterAccount_DisabledSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{Name: "disabled-test", AuthType: models.AuthTypeClientCredentials, AuthInfo: string(authJSON)}
	err := database.Accounts.Create(ctx, &account)
	require.NoError(t, err)

	schedule := models.Schedule{AccountID: account.ID, Enabled: false}
	err = database.Schedules.Create(ctx, &schedule)
	require.NoError(t, err)

	// Should not panic, should remove timer
	sched.RegisterAccount(ctx, account.ID)
}

func TestUnregisterAccount(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Should not panic even if account doesn't exist
	sched.UnregisterAccount(999)
}

func TestRegisterAccount_NonexistentSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Schedule doesn't exist - should not panic
	sched.RegisterAccount(ctx, 9999)
}

func TestTriggerNow(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "trigger-token",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{Name: "trigger-test", AuthType: models.AuthTypeAuthCode, AuthInfo: string(authJSON)}
	err := database.Accounts.Create(ctx, &account)
	require.NoError(t, err)

	result, err := sched.TriggerNow(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.TaskLog)
}

func TestStartWithActiveSchedules(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()

	// Create account with active schedule
	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{Name: "active-test", AuthType: models.AuthTypeAuthCode, AuthInfo: string(authJSON)}
	err := database.Accounts.Create(ctx, &account)
	require.NoError(t, err)

	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	err = database.Schedules.Create(ctx, &schedule)
	require.NoError(t, err)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	sched.Stop()
}

// --------------- timeWeight tests ---------------

func TestTimeWeight_WeekdayBusinessHours(t *testing.T) {
	// Tuesday 10am (weekday, business hours 9-18) -> 0.5
	ts := time.Date(2025, 6, 10, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, 0.5, scheduler.TimeWeight(ts))
}

func TestTimeWeight_WeekdayEvening(t *testing.T) {
	// Tuesday 20:00 (weekday, evening 18-23) -> 1.0
	ts := time.Date(2025, 6, 10, 20, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.0, scheduler.TimeWeight(ts))
}

func TestTimeWeight_WeekdayNight(t *testing.T) {
	// Tuesday 3am (weekday, night 23-9) -> 2.0
	ts := time.Date(2025, 6, 10, 3, 0, 0, 0, time.UTC)
	assert.Equal(t, 2.0, scheduler.TimeWeight(ts))
}

func TestTimeWeight_WeekdayLateNight(t *testing.T) {
	// Wednesday 23:30 -> 2.0 (>= 23)
	ts := time.Date(2025, 6, 11, 23, 30, 0, 0, time.UTC)
	assert.Equal(t, 2.0, scheduler.TimeWeight(ts))
}

func TestTimeWeight_Weekend(t *testing.T) {
	// Saturday any hour -> 1.5
	ts := time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.5, scheduler.TimeWeight(ts))
}

func TestTimeWeight_Sunday(t *testing.T) {
	ts := time.Date(2025, 6, 15, 3, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.5, scheduler.TimeWeight(ts))
}

func TestTimeWeight_BoundaryHour9(t *testing.T) {
	// Exactly 9:00 on weekday -> business hours -> 0.5
	ts := time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC)
	assert.Equal(t, 0.5, scheduler.TimeWeight(ts))
}

func TestTimeWeight_BoundaryHour18(t *testing.T) {
	// Exactly 18:00 on weekday -> evening -> 1.0
	ts := time.Date(2025, 6, 10, 18, 0, 0, 0, time.UTC)
	assert.Equal(t, 1.0, scheduler.TimeWeight(ts))
}

// --------------- ComputeNextRun with RealisticTiming ---------------

func TestComputeNextRun_RealisticTiming_Weekday(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	cfg := config.Get()
	cfg.Scheduler.RealisticTiming = true
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2 // fixed so Intn is never called

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	// Tuesday 10am -> business hours weight 0.5
	fixedNow := time.Date(2025, 6, 10, 10, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	diff := next.Sub(fixedNow)
	// hours = 2 * 0.5 = 1.0
	assert.InDelta(t, 1*3600, diff.Seconds(), 1)
}

func TestComputeNextRun_RealisticTiming_Weekend(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	cfg := config.Get()
	cfg.Scheduler.RealisticTiming = true
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	// Saturday -> weight 1.5
	fixedNow := time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	diff := next.Sub(fixedNow)
	// hours = 2 * 1.5 = 3.0
	assert.InDelta(t, 3*3600, diff.Seconds(), 1)
}

func TestComputeNextRun_RealisticTiming_Night(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	cfg := config.Get()
	cfg.Scheduler.RealisticTiming = true
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	// Tuesday 3am -> night weight 2.0
	fixedNow := time.Date(2025, 6, 10, 3, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	diff := next.Sub(fixedNow)
	// hours = 2 * 2.0 = 4.0
	assert.InDelta(t, 4*3600, diff.Seconds(), 1)
}

func TestComputeNextRun_MinHoursZeroDefault(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	cfg := config.Get()
	cfg.Scheduler.MinHours = 0
	cfg.Scheduler.MaxHours = 0
	cfg.Scheduler.RealisticTiming = false

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	next := sched.ComputeNextRun()
	diff := next.Sub(fixedNow)
	// minHours defaults to 2, maxH < minH so maxH = minH = 2
	assert.InDelta(t, 2*3600, diff.Seconds(), 1)
}

// --------------- RunTask tests ---------------

func newTestServers() (*httptest.Server, *httptest.Server) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "test-access-token",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":[]}`))
	}))
	return tokenServer, graphServer
}

func createTestAccount(t *testing.T, ctx context.Context, name string, authType string) models.Account {
	t.Helper()
	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{Name: name, AuthType: authType, AuthInfo: string(authJSON)}
	err := database.Accounts.Create(ctx, &account)
	require.NoError(t, err)
	return account
}

func taskLogCount(t *testing.T, ctx context.Context, accountID uint) int {
	t.Helper()
	logs, _, err := database.TaskLogs.ListWithTotal(ctx, database.TaskLogFilter{AccountID: accountID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	return len(logs)
}

func TestRunTask_Success(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "runtask-ok", models.AuthTypeAuthCode)

	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }
	sched.Start(ctx)
	defer sched.Stop()

	sched.RunTask(account.ID)

	// Verify schedule was updated with next_run_at and last_run_at
	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedSched.LastRunAt)
	assert.NotNil(t, updatedSched.NextRunAt)
	assert.False(t, updatedSched.Paused)
}

func TestRunTask_ContextCancelled(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)
	cancel() // Cancel immediately

	// RunTask should return early when context is cancelled
	// Should not panic
	sched.RunTask(999)
	sched.Stop()
}

func TestRunTask_AccountNotFound(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Account ID 9999 doesn't exist, should not panic
	sched.RunTask(9999)
}

func TestRunTask_ScheduleNotFound(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	account := createTestAccount(t, ctx, "no-schedule", models.AuthTypeAuthCode)

	sched.Start(ctx)
	defer sched.Stop()

	// Account exists but no schedule - should not panic
	sched.RunTask(account.ID)
}

func TestRunTask_DisabledSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	account := createTestAccount(t, ctx, "disabled-runtask", models.AuthTypeAuthCode)
	schedule := models.Schedule{AccountID: account.ID, Enabled: false, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	sched.Start(ctx)
	defer sched.Stop()

	// Should return early because schedule is disabled
	sched.RunTask(account.ID)

	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.Nil(t, updatedSched.LastRunAt) // Should not have run
}

func TestRunTask_PausedSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	account := createTestAccount(t, ctx, "paused-runtask", models.AuthTypeAuthCode)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: true}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	sched.Start(ctx)
	defer sched.Stop()

	sched.RunTask(account.ID)

	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.Nil(t, updatedSched.LastRunAt) // Should not have run
}

func TestRunTask_AutoPause(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	// Token server that always fails
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:   "Bearer",
			AccessToken: "test-token",
			ExpiresIn:   3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	// Graph server that always fails
	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"code":"Authorization_RequestDenied"}}`))
	}))
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "autopause-test", models.AuthTypeClientCredentials)

	// Set PauseThreshold to 50%
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false, PauseThreshold: 50}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }
	sched.Start(ctx)
	defer sched.Stop()

	// Run task multiple times to build up failure history
	for i := 0; i < 3; i++ {
		// Re-enable if paused (so we keep accumulating history)
		_ = database.Schedules.UpdateFields(ctx, account.ID, map[string]any{"paused": false, "enabled": true})
		sched.RunTask(account.ID)
	}

	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.True(t, updatedSched.Paused, "account should be auto-paused due to low health")
}

func TestRunTask_TokenFailure_NotifiesAllFailed(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	// Token server that fails
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"bad token"}`))
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "token-fail-test", models.AuthTypeAuthCode)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	// Set up notification config with on_task_all_failed enabled
	notifyCfg := models.NotificationConfig{
		URL:             "generic://example.com",
		OnTaskAllFailed: true,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	// Set notifier to nil so we don't actually send notifications, but cover the code path
	sched.Notifier = nil
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }
	sched.Start(ctx)
	defer sched.Stop()

	// Should not panic - token failure triggers notification path
	sched.RunTask(account.ID)
}

func TestRunTask_AllEndpointsFail_NotifiesAllFailed(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:   "Bearer",
			AccessToken: "test-token",
			ExpiresIn:   3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"code":"Forbidden"}}`))
	}))
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "all-fail-notify", models.AuthTypeClientCredentials)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Notifier = nil
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }
	sched.Start(ctx)
	defer sched.Stop()

	sched.RunTask(account.ID)

	// Verify task log was created
	logs, _, err := database.TaskLogs.ListWithTotal(ctx, database.TaskLogFilter{AccountID: account.ID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, logs)
}

// --------------- computeHealth tests ---------------

func TestComputeHealth_NoLogs(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// No task logs for account 999
	health := sched.ComputeHealth(999)
	assert.Equal(t, float64(-1), health)
}

func TestComputeHealth_AllSuccess(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	account := createTestAccount(t, ctx, "health-ok", models.AuthTypeAuthCode)

	// Create task logs with all success
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		finished := now
		tl := models.TaskLog{
			AccountID:      account.ID,
			RunID:          fmt.Sprintf("run-health-ok-%d", i),
			TriggerType:    models.TriggerScheduled,
			TotalEndpoints: 5,
			SuccessCount:   5,
			FailCount:      0,
			StartedAt:      now,
			FinishedAt:     &finished,
		}
		require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))
	}

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Start(ctx)
	defer sched.Stop()

	health := sched.ComputeHealth(account.ID)
	assert.InDelta(t, 100.0, health, 0.1)
}

func TestComputeHealth_Partial(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	account := createTestAccount(t, ctx, "health-partial", models.AuthTypeAuthCode)

	// Health is computed as fraction of task logs with FailCount==0.
	// Create 2 successful logs (FailCount=0) and 2 failed logs (FailCount>0) → 50% health.
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		finished := now
		tl := models.TaskLog{
			AccountID:      account.ID,
			RunID:          fmt.Sprintf("run-health-ok-%d", i),
			TriggerType:    models.TriggerScheduled,
			TotalEndpoints: 5,
			SuccessCount:   5,
			FailCount:      0,
			StartedAt:      now,
			FinishedAt:     &finished,
		}
		require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))
	}
	for i := 0; i < 2; i++ {
		finished := now
		tl := models.TaskLog{
			AccountID:      account.ID,
			RunID:          fmt.Sprintf("run-health-fail-%d", i),
			TriggerType:    models.TriggerScheduled,
			TotalEndpoints: 5,
			SuccessCount:   0,
			FailCount:      5,
			StartedAt:      now,
			FinishedAt:     &finished,
		}
		require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))
	}

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Start(ctx)
	defer sched.Stop()

	health := sched.ComputeHealth(account.ID)
	assert.InDelta(t, 50.0, health, 0.1)
}

func TestComputeHealth_ZeroEndpoints(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	account := createTestAccount(t, ctx, "health-zero-ep", models.AuthTypeAuthCode)

	// A task log with TotalEndpoints=0 and FailCount=0 counts as a fully successful run.
	// Health = 1/1 * 100 = 100 (all runs have FailCount==0).
	now := time.Now().UTC()
	finished := now
	tl := models.TaskLog{
		AccountID:      account.ID,
		RunID:          "run-health-zero-ep",
		TriggerType:    models.TriggerScheduled,
		TotalEndpoints: 0,
		SuccessCount:   0,
		FailCount:      0,
		StartedAt:      now,
		FinishedAt:     &finished,
	}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Start(ctx)
	defer sched.Stop()

	// FailCount == 0 → counted as success → health = 100%
	health := sched.ComputeHealth(account.ID)
	assert.InDelta(t, 100.0, health, 0.1)
}

// --------------- checkAuthExpiry tests ---------------

func TestCheckAuthExpiry_NoNotificationConfig(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// No notification config -> should not panic
	sched.CheckAuthExpiry()
}

func TestCheckAuthExpiry_Disabled(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:          "generic://example.com",
		OnAuthExpiry: false,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	defer sched.Stop()

	// on_auth_expiry is false, should return early
	sched.CheckAuthExpiry()
}

func TestCheckAuthExpiry_FindsExpiringAccounts(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()

	notifyCfg := models.NotificationConfig{
		URL:              "generic://example.com",
		OnAuthExpiry:     true,
		ExpiryDaysBefore: 7,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)

	// Create account expiring in 3 days (within threshold)
	expiresAt := fixedNow.AddDate(0, 0, 3)
	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{
		Name:          "expiring-soon",
		AuthType:      models.AuthTypeAuthCode,
		AuthInfo:      string(authJSON),
		AuthExpiresAt: &expiresAt,
	}
	require.NoError(t, database.Accounts.Create(ctx, &account))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }
	sched.Notifier = nil // Don't actually send notifications
	sched.Start(ctx)
	defer sched.Stop()

	// Should find the expiring account and attempt notification (but Notifier is nil)
	sched.CheckAuthExpiry()
}

func TestCheckAuthExpiry_ExpiredAccount(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()

	notifyCfg := models.NotificationConfig{
		URL:              "generic://example.com",
		OnAuthExpiry:     true,
		ExpiryDaysBefore: 7,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)

	// Account already expired 2 days ago
	expiresAt := fixedNow.AddDate(0, 0, -2)
	authInfo := models.AuthInfoData{ClientID: "c", ClientSecret: "s", TenantID: "t", RefreshToken: "r"}
	authJSON, _ := json.Marshal(authInfo)
	account := models.Account{
		Name:          "already-expired",
		AuthType:      models.AuthTypeAuthCode,
		AuthInfo:      string(authJSON),
		AuthExpiresAt: &expiresAt,
	}
	require.NoError(t, database.Accounts.Create(ctx, &account))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }
	sched.Notifier = nil
	sched.Start(ctx)
	defer sched.Stop()

	// Should process the already-expired account
	sched.CheckAuthExpiry()
}

func TestCheckAuthExpiry_DefaultDaysBefore(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()

	// ExpiryDaysBefore is 0, should default to 7
	notifyCfg := models.NotificationConfig{
		URL:              "generic://example.com",
		OnAuthExpiry:     true,
		ExpiryDaysBefore: 0,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }
	sched.Notifier = nil
	sched.Start(ctx)
	defer sched.Stop()

	sched.CheckAuthExpiry()
}

// --------------- notifyIfEnabled tests ---------------

func TestNotifyIfEnabled_NilNotifier(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	sched.Notifier = nil

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Should return early without panic
	sched.NotifyIfEnabled("on_auth_expiry", "title", "message")
}

func TestNotifyIfEnabled_NoConfig(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// No notification setting in DB -> should not panic
	sched.NotifyIfEnabled("on_auth_expiry", "title", "message")
}

func TestNotifyIfEnabled_EmptyURL(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:          "",
		OnAuthExpiry: true,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	defer sched.Stop()

	// URL is empty -> should return early
	sched.NotifyIfEnabled("on_auth_expiry", "title", "message")
}

func TestNotifyIfEnabled_EventDisabled(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:             "generic://example.com",
		OnAuthExpiry:    false,
		OnTaskAllFailed: false,
		OnHealthLow:     false,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	defer sched.Stop()

	// All events disabled
	sched.NotifyIfEnabled("on_auth_expiry", "title", "message")
	sched.NotifyIfEnabled("on_task_all_failed", "title", "message")
	sched.NotifyIfEnabled("on_health_low", "title", "message")
}

func TestNotifyIfEnabled_EventEnabled_SendFails(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:          "generic://invalid-url-that-fails",
		OnAuthExpiry: true,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	// Use real notifier so Send is called but may error

	sched.Start(ctx)
	defer sched.Stop()

	// The real notifier Send will fail on invalid URL, but should not panic
	sched.NotifyIfEnabled("on_auth_expiry", "test title", "test message")
}

func TestNotifyIfEnabled_OnTaskAllFailed(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:             "generic://invalid-url-that-fails",
		OnTaskAllFailed: true,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	defer sched.Stop()

	sched.NotifyIfEnabled("on_task_all_failed", "test title", "test message")
}

func TestNotifyIfEnabled_OnHealthLow(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()
	notifyCfg := models.NotificationConfig{
		URL:         "generic://invalid-url-that-fails",
		OnHealthLow: true,
	}
	notifyJSON, _ := json.Marshal(notifyCfg)
	require.NoError(t, database.Settings.Upsert(ctx, models.SettingKeyNotification, string(notifyJSON)))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	defer sched.Stop()

	sched.NotifyIfEnabled("on_health_low", "test title", "test message")
}

// --------------- scheduleAccount / timer callback ---------------

func TestStart_FutureNextRunAtWaitsUntilScheduledTime(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "future-nextrun", models.AuthTypeAuthCode)

	fixedNow := time.Now().UTC()
	futureTime := fixedNow.Add(250 * time.Millisecond)
	schedule := models.Schedule{
		AccountID: account.ID,
		Enabled:   true,
		Paused:    false,
		NextRunAt: &futureTime,
	}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }

	sched.Start(ctx)
	defer sched.Stop()

	require.Never(t, func() bool {
		return taskLogCount(t, ctx, account.ID) > 0
	}, 100*time.Millisecond, 25*time.Millisecond, "future next run should not fire early")

	require.Eventually(t, func() bool {
		return taskLogCount(t, ctx, account.ID) == 1
	}, 2*time.Second, 25*time.Millisecond)
}

func TestStart_MissingNextRunAtComputesNextNormalRun(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	cfg := config.Get()
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2
	cfg.Scheduler.RealisticTiming = false

	ctx := context.Background()
	account := createTestAccount(t, ctx, "missing-nextrun", models.AuthTypeAuthCode)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())
	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)
	sched.Now = func() time.Time { return fixedNow }

	sched.Start(ctx)
	defer sched.Stop()

	stored, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.NextRunAt)
	assert.WithinDuration(t, fixedNow.Add(2*time.Hour), *stored.NextRunAt, time.Second)
	assert.Nil(t, stored.LastRunAt)
	assert.Equal(t, 0, taskLogCount(t, ctx, account.ID), "missing next run should not trigger immediate execution")
}

func TestStart_PastNextRunAtTriggersImmediateCatchUp(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	cfg := config.Get()
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2
	cfg.Scheduler.RealisticTiming = false

	ctx := context.Background()
	account := createTestAccount(t, ctx, "past-nextrun", models.AuthTypeAuthCode)

	fixedNow := time.Now().UTC()
	pastTime := fixedNow.Add(-1 * time.Hour)
	schedule := models.Schedule{
		AccountID: account.ID,
		Enabled:   true,
		Paused:    false,
		NextRunAt: &pastTime,
	}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }

	sched.Start(ctx)
	defer sched.Stop()

	require.Eventually(t, func() bool {
		return taskLogCount(t, ctx, account.ID) == 1
	}, 2*time.Second, 25*time.Millisecond)
}

func TestStart_PastNextRunAtCatchesUpOnlyOnceAndPersistsFreshNextRun(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	cfg := config.Get()
	cfg.Scheduler.MinHours = 2
	cfg.Scheduler.MaxHours = 2
	cfg.Scheduler.RealisticTiming = false

	ctx := context.Background()
	account := createTestAccount(t, ctx, "past-nextrun-once", models.AuthTypeAuthCode)

	fixedNow := time.Now().UTC()
	pastTime := fixedNow.Add(-3 * time.Hour)
	schedule := models.Schedule{
		AccountID: account.ID,
		Enabled:   true,
		Paused:    false,
		NextRunAt: &pastTime,
	}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }

	sched.Start(ctx)
	defer sched.Stop()

	require.Eventually(t, func() bool {
		return taskLogCount(t, ctx, account.ID) == 1
	}, 2*time.Second, 25*time.Millisecond)

	stored, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.NextRunAt)
	assert.True(t, stored.NextRunAt.After(fixedNow), "catch-up should persist a fresh future next run")
	assert.WithinDuration(t, fixedNow.Add(2*time.Hour), *stored.NextRunAt, time.Second)

	require.Never(t, func() bool {
		return taskLogCount(t, ctx, account.ID) > 1
	}, 300*time.Millisecond, 25*time.Millisecond, "overdue startup path should catch up exactly once")
}

func TestRegisterAccount_PausedSchedule(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	account := createTestAccount(t, ctx, "paused-reg", models.AuthTypeAuthCode)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: true}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	// Paused schedule -> removes timer
	sched.RegisterAccount(ctx, account.ID)
}

// --------------- Timer fires and runs actual task ---------------

func TestTimerCallback_FiresAndExecutes(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "timer-fire", models.AuthTypeAuthCode)

	// Set NextRunAt to 50ms after the fixed scheduler clock so the timer fires quickly.
	fixedNow := time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC)
	nearFuture := fixedNow.Add(50 * time.Millisecond)
	schedule := models.Schedule{
		AccountID: account.ID,
		Enabled:   true,
		Paused:    false,
		NextRunAt: &nearFuture,
	}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return fixedNow }

	sched.Start(ctx)
	defer sched.Stop()

	require.Eventually(t, func() bool {
		logs, _, err := database.TaskLogs.ListWithTotal(ctx, database.TaskLogFilter{AccountID: account.ID, Page: 1, PageSize: 10})
		require.NoError(t, err)
		return len(logs) > 0
	}, 5*time.Second, 100*time.Millisecond, "timer should have fired and created a task log")
}

func TestTimerCallback_ReschedulesAfterSuccess(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer, graphServer := newTestServers()
	defer tokenServer.Close()
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "timer-resched", models.AuthTypeAuthCode)

	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }

	sched.Start(ctx)
	defer sched.Stop()

	sched.RunTask(account.ID)

	// After successful run, NextRunAt should be set
	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedSched.NextRunAt)
}

// --------------- TriggerNow error case ---------------

func TestTriggerNow_AccountNotFound(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	_, err := sched.TriggerNow(ctx, 9999)
	assert.Error(t, err)
}

// --------------- Multiple accounts start ---------------

func TestStartWithMultipleActiveSchedules(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		account := createTestAccount(t, ctx, fmt.Sprintf("multi-%d", i), models.AuthTypeAuthCode)
		schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
		require.NoError(t, database.Schedules.Create(ctx, &schedule))
	}

	exec := executor.New(oauth.NewService(nil), newRng())
	sched := scheduler.New(exec, newRng())

	sched.Start(ctx)
	sched.Stop()
}

func TestRunTask_ClientCredentials(t *testing.T) {
	initTestDB(t)
	initTestConfig(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:   "Bearer",
			AccessToken: "cc-token",
			ExpiresIn:   3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":[]}`))
	}))
	defer graphServer.Close()

	ctx := context.Background()
	account := createTestAccount(t, ctx, "cc-runtask", models.AuthTypeClientCredentials)
	schedule := models.Schedule{AccountID: account.ID, Enabled: true, Paused: false}
	require.NoError(t, database.Schedules.Create(ctx, &schedule))

	exec := makeExecutor(tokenServer.URL, graphServer.URL)
	sched := scheduler.New(exec, newRng())
	sched.Now = func() time.Time { return time.Date(2025, 6, 10, 14, 0, 0, 0, time.UTC) }
	sched.Start(ctx)
	defer sched.Stop()

	sched.RunTask(account.ID)

	updatedSched, err := database.Schedules.GetByAccountID(ctx, account.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedSched.LastRunAt)
}

// rewriteTransport redirects all requests to a test server.
type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.target[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}
