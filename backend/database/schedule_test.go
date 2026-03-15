package database

import (
	"testing"
	"time"

	"e5-renewal/backend/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestScheduleRepo_Create_And_GetByAccountID(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "sched-acc", models.AuthTypeClientCredentials)

	nextRun := time.Now().Add(2 * time.Hour)
	sched := &models.Schedule{
		AccountID:      acc.ID,
		Enabled:        true,
		PauseThreshold: 50,
		NextRunAt:      &nextRun,
	}
	require.NoError(t, Schedules.Create(ctx, sched))
	require.NotZero(t, sched.ID)

	fetched, err := Schedules.GetByAccountID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, acc.ID, fetched.AccountID)
	assert.True(t, fetched.Enabled)
	assert.Equal(t, 50, fetched.PauseThreshold)
	assert.NotNil(t, fetched.NextRunAt)
}

func TestScheduleRepo_GetByAccountID_NotFound(t *testing.T) {
	ctx := setupTestDB(t)

	_, err := Schedules.GetByAccountID(ctx, 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestScheduleRepo_Save(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "sched-save", models.AuthTypeClientCredentials)

	sched := &models.Schedule{AccountID: acc.ID, Enabled: true}
	require.NoError(t, Schedules.Create(ctx, sched))

	sched.Paused = true
	sched.PauseReason = "health too low"
	require.NoError(t, Schedules.Save(ctx, sched))

	fetched, err := Schedules.GetByAccountID(ctx, acc.ID)
	require.NoError(t, err)
	assert.True(t, fetched.Paused)
	assert.Equal(t, "health too low", fetched.PauseReason)
}

func TestScheduleRepo_DeleteByAccountID(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "sched-del", models.AuthTypeClientCredentials)

	sched := &models.Schedule{AccountID: acc.ID, Enabled: true}
	require.NoError(t, Schedules.Create(ctx, sched))

	require.NoError(t, Schedules.DeleteByAccountID(ctx, acc.ID))

	_, err := Schedules.GetByAccountID(ctx, acc.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestScheduleRepo_ListActive(t *testing.T) {
	ctx := setupTestDB(t)

	acc1 := createTestAccount(t, ctx, "active-1", models.AuthTypeClientCredentials)
	acc2 := createTestAccount(t, ctx, "active-2", models.AuthTypeClientCredentials)
	acc3 := createTestAccount(t, ctx, "disabled", models.AuthTypeClientCredentials)
	acc4 := createTestAccount(t, ctx, "paused", models.AuthTypeClientCredentials)

	require.NoError(t, Schedules.Create(ctx, &models.Schedule{AccountID: acc1.ID, Enabled: true}))
	require.NoError(t, Schedules.Create(ctx, &models.Schedule{AccountID: acc2.ID, Enabled: true}))
	require.NoError(t, Schedules.Create(ctx, &models.Schedule{AccountID: acc3.ID, Enabled: false}))
	require.NoError(t, Schedules.Create(ctx, &models.Schedule{AccountID: acc4.ID, Enabled: true, Paused: true}))

	active, err := Schedules.ListActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 2)
}

func TestScheduleRepo_ListActive_Empty(t *testing.T) {
	ctx := setupTestDB(t)

	active, err := Schedules.ListActive(ctx)
	require.NoError(t, err)
	assert.Empty(t, active)
}

func TestScheduleRepo_UpdateFields(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "update-fields", models.AuthTypeClientCredentials)

	sched := &models.Schedule{AccountID: acc.ID, Enabled: true}
	require.NoError(t, Schedules.Create(ctx, sched))

	now := time.Now()
	nextRun := now.Add(4 * time.Hour)
	require.NoError(t, Schedules.UpdateFields(ctx, acc.ID, map[string]any{
		"last_run_at":  now,
		"next_run_at":  nextRun,
		"paused":       true,
		"pause_reason": "auto-pause",
	}))

	fetched, err := Schedules.GetByAccountID(ctx, acc.ID)
	require.NoError(t, err)
	assert.True(t, fetched.Paused)
	assert.Equal(t, "auto-pause", fetched.PauseReason)
	assert.NotNil(t, fetched.LastRunAt)
	assert.NotNil(t, fetched.NextRunAt)
}
