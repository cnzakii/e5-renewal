package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"e5-renewal/backend/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createTestTaskLog(t *testing.T, ctx context.Context, accountID uint, runID string, success, fail int, startedAt time.Time) *models.TaskLog {
	t.Helper()
	tl := &models.TaskLog{
		AccountID:      accountID,
		RunID:          runID,
		TriggerType:    models.TriggerScheduled,
		TotalEndpoints: success + fail,
		SuccessCount:   success,
		FailCount:      fail,
		StartedAt:      startedAt,
	}
	require.NoError(t, GetDB(ctx).Create(tl).Error)
	return tl
}

func TestTaskLogRepo_CreateWithEndpoints(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "tl-acc", models.AuthTypeClientCredentials)

	taskLog := &models.TaskLog{
		AccountID:      acc.ID,
		RunID:          "run-1",
		TriggerType:    models.TriggerScheduled,
		TotalEndpoints: 3,
		SuccessCount:   2,
		FailCount:      1,
		StartedAt:      time.Now(),
	}
	endpoints := []models.EndpointLog{
		{EndpointName: "/me", Scope: "User.Read", HTTPStatus: 200, Success: true, ExecutedAt: time.Now()},
		{EndpointName: "/users", Scope: "User.ReadAll", HTTPStatus: 200, Success: true, ExecutedAt: time.Now()},
		{EndpointName: "/drive", Scope: "Files.Read", HTTPStatus: 403, Success: false, ErrorMessage: "forbidden", ExecutedAt: time.Now()},
	}

	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, taskLog, endpoints))
	require.NotZero(t, taskLog.ID)

	// All endpoint logs should have the task log ID set.
	for _, ep := range endpoints {
		assert.Equal(t, taskLog.ID, ep.TaskLogID)
	}

	// Verify via EndpointLogs repo.
	fetched, err := EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
	require.NoError(t, err)
	assert.Len(t, fetched, 3)
}

func TestTaskLogRepo_ListWithTotal_NoFilter(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "lwt-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	for i := 0; i < 5; i++ {
		createTestTaskLog(t, ctx, acc.ID, fmt.Sprintf("lwt-run-%d", i), 3, 0, now.Add(time.Duration(-i)*time.Hour))
	}

	logs, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{Page: 1, PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 3)
	// Should be ordered by started_at desc.
	assert.True(t, logs[0].StartedAt.After(logs[1].StartedAt) || logs[0].StartedAt.Equal(logs[1].StartedAt))
}

func TestTaskLogRepo_ListWithTotal_FilterByAccountID(t *testing.T) {
	ctx := setupTestDB(t)
	acc1 := createTestAccount(t, ctx, "lwt-a1", models.AuthTypeClientCredentials)
	acc2 := createTestAccount(t, ctx, "lwt-a2", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc1.ID, "a1-run-1", 3, 0, now)
	createTestTaskLog(t, ctx, acc1.ID, "a1-run-2", 3, 0, now.Add(-time.Hour))
	createTestTaskLog(t, ctx, acc2.ID, "a2-run-1", 3, 0, now)

	logs, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{AccountID: acc1.ID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)
}

func TestTaskLogRepo_ListWithTotal_FilterByStatus(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "status-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "success-run", 3, 0, now)                  // success
	createTestTaskLog(t, ctx, acc.ID, "partial-run", 2, 1, now.Add(-time.Hour))  // partial
	createTestTaskLog(t, ctx, acc.ID, "failed-run", 0, 3, now.Add(-2*time.Hour)) // failed

	tests := []struct {
		status string
		want   int64
	}{
		{"success", 1},
		{"partial", 1},
		{"failed", 1},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			_, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{Status: tt.status, Page: 1, PageSize: 10})
			require.NoError(t, err)
			assert.Equal(t, tt.want, total)
		})
	}
}

func TestTaskLogRepo_ListWithTotal_FilterByTriggerType(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "trigger-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "sched-run", 3, 0, now)

	manual := &models.TaskLog{
		AccountID: acc.ID, RunID: "manual-run", TriggerType: models.TriggerManual,
		TotalEndpoints: 1, SuccessCount: 1, StartedAt: now,
	}
	require.NoError(t, GetDB(ctx).Create(manual).Error)

	_, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{TriggerType: "scheduled", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)

	_, total, err = TaskLogs.ListWithTotal(ctx, TaskLogFilter{TriggerType: "manual", Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestTaskLogRepo_ListWithTotal_FilterByDate(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "date-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	createTestTaskLog(t, ctx, acc.ID, "today-run", 1, 0, now)
	createTestTaskLog(t, ctx, acc.ID, "yesterday-run", 1, 0, yesterday)
	createTestTaskLog(t, ctx, acc.ID, "old-run", 1, 0, twoDaysAgo)

	from := yesterday.Add(-time.Minute)
	to := yesterday.Add(time.Minute)
	_, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{DateFrom: &from, DateTo: &to, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestTaskLogRepo_ListWithTotal_FilterByID(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "id-acc", models.AuthTypeClientCredentials)

	tl := createTestTaskLog(t, ctx, acc.ID, "id-run", 1, 0, time.Now())

	logs, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{ID: tl.ID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, tl.ID, logs[0].ID)
}

func TestTaskLogRepo_CountByAccount(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "count-acc", models.AuthTypeClientCredentials)

	count, err := TaskLogs.CountByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Zero(t, count)

	createTestTaskLog(t, ctx, acc.ID, "cnt-1", 3, 0, time.Now())
	createTestTaskLog(t, ctx, acc.ID, "cnt-2", 2, 1, time.Now())

	count, err = TaskLogs.CountByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestTaskLogRepo_CountSuccessByAccount(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "succ-acc", models.AuthTypeClientCredentials)

	createTestTaskLog(t, ctx, acc.ID, "succ-1", 3, 0, time.Now()) // success (fail_count=0)
	createTestTaskLog(t, ctx, acc.ID, "succ-2", 2, 1, time.Now()) // partial
	createTestTaskLog(t, ctx, acc.ID, "succ-3", 0, 3, time.Now()) // failed

	count, err := TaskLogs.CountSuccessByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestTaskLogRepo_LastByAccount(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "last-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "old-run", 1, 0, now.Add(-2*time.Hour))
	createTestTaskLog(t, ctx, acc.ID, "new-run", 1, 0, now)

	last, err := TaskLogs.LastByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-run", last.RunID)
}

func TestTaskLogRepo_LastByAccount_NotFound(t *testing.T) {
	ctx := setupTestDB(t)

	_, err := TaskLogs.LastByAccount(ctx, 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestTaskLogRepo_RecentN(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "recent-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	for i := 0; i < 10; i++ {
		createTestTaskLog(t, ctx, acc.ID, fmt.Sprintf("recent-%d", i), 1, 0, now.Add(time.Duration(-i)*time.Hour))
	}

	logs, err := TaskLogs.RecentN(ctx, 3)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
	// Most recent first.
	assert.Equal(t, "recent-0", logs[0].RunID)
}

func TestTaskLogRepo_Last20ByAccount(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "l20-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	for i := 0; i < 25; i++ {
		createTestTaskLog(t, ctx, acc.ID, fmt.Sprintf("l20-%d", i), 1, 0, now.Add(time.Duration(-i)*time.Hour))
	}

	logs, err := TaskLogs.Last20ByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Len(t, logs, 20)
}

func TestTaskLogRepo_CountInPeriod(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "cip-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "cip-1", 1, 0, now)
	createTestTaskLog(t, ctx, acc.ID, "cip-2", 1, 0, now.Add(-48*time.Hour))

	count, err := TaskLogs.CountInPeriod(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Zero time should count all.
	count, err = TaskLogs.CountInPeriod(ctx, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestTaskLogRepo_CountErrorsInPeriod(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ceip-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "ceip-1", 3, 0, now)                    // no errors
	createTestTaskLog(t, ctx, acc.ID, "ceip-2", 2, 1, now)                    // has errors
	createTestTaskLog(t, ctx, acc.ID, "ceip-3", 0, 3, now.Add(-48*time.Hour)) // old with errors

	count, err := TaskLogs.CountErrorsInPeriod(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = TaskLogs.CountErrorsInPeriod(ctx, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestTaskLogRepo_EndpointCountsByAccount(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "epc-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "epc-1", 3, 1, now) // total=4, success=3
	createTestTaskLog(t, ctx, acc.ID, "epc-2", 2, 0, now) // total=2, success=2

	totalEp, successEp, err := TaskLogs.EndpointCountsByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(6), totalEp)
	assert.Equal(t, int64(5), successEp)
}

func TestTaskLogRepo_EndpointCountsByAccount_NoLogs(t *testing.T) {
	ctx := setupTestDB(t)

	totalEp, successEp, err := TaskLogs.EndpointCountsByAccount(ctx, 9999)
	require.NoError(t, err)
	assert.Zero(t, totalEp)
	assert.Zero(t, successEp)
}

func TestTaskLogRepo_EndpointCountsInPeriod(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ecip-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "ecip-1", 5, 2, now)                    // total=7, success=5
	createTestTaskLog(t, ctx, acc.ID, "ecip-2", 3, 0, now.Add(-48*time.Hour)) // old

	totalEp, successEp, err := TaskLogs.EndpointCountsInPeriod(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(7), totalEp)
	assert.Equal(t, int64(5), successEp)

	// Zero time counts all.
	totalEp, successEp, err = TaskLogs.EndpointCountsInPeriod(ctx, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, int64(10), totalEp)
	assert.Equal(t, int64(8), successEp)
}

func TestTaskLogRepo_FindInTimeRange(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ftr-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	createTestTaskLog(t, ctx, acc.ID, "ftr-1", 1, 0, now)
	createTestTaskLog(t, ctx, acc.ID, "ftr-2", 1, 0, now.Add(-2*time.Hour))
	createTestTaskLog(t, ctx, acc.ID, "ftr-3", 1, 0, now.Add(-48*time.Hour))

	logs, err := TaskLogs.FindInTimeRange(ctx, now.Add(-24*time.Hour), now.Add(time.Hour))
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestTaskLogRepo_ListWithTotal_Pagination(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "pag-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	for i := 0; i < 7; i++ {
		createTestTaskLog(t, ctx, acc.ID, fmt.Sprintf("pag-%d", i), 1, 0, now.Add(time.Duration(-i)*time.Hour))
	}

	// Page 1.
	logs, total, err := TaskLogs.ListWithTotal(ctx, TaskLogFilter{Page: 1, PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, int64(7), total)
	assert.Len(t, logs, 3)

	// Page 2.
	logs, total, err = TaskLogs.ListWithTotal(ctx, TaskLogFilter{Page: 2, PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, int64(7), total)
	assert.Len(t, logs, 3)

	// Page 3 (last, partial).
	logs, total, err = TaskLogs.ListWithTotal(ctx, TaskLogFilter{Page: 3, PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, int64(7), total)
	assert.Len(t, logs, 1)
}
