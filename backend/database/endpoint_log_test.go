package database

import (
	"testing"
	"time"

	"e5-renewal/backend/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointLogRepo_ListByTaskLogID(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ep-acc", models.AuthTypeClientCredentials)

	now := time.Now()
	taskLog := &models.TaskLog{
		AccountID:      acc.ID,
		RunID:          "ep-run-1",
		TriggerType:    models.TriggerScheduled,
		TotalEndpoints: 3,
		SuccessCount:   2,
		FailCount:      1,
		StartedAt:      now,
	}
	endpoints := []models.EndpointLog{
		{EndpointName: "/me", Scope: "User.Read", HTTPStatus: 200, Success: true, ExecutedAt: now.Add(1 * time.Second)},
		{EndpointName: "/drive", Scope: "Files.Read", HTTPStatus: 200, Success: true, ExecutedAt: now.Add(2 * time.Second)},
		{EndpointName: "/mail", Scope: "Mail.Read", HTTPStatus: 403, Success: false, ErrorMessage: "Insufficient privileges", ExecutedAt: now.Add(3 * time.Second)},
	}
	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, taskLog, endpoints))

	fetched, err := EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
	require.NoError(t, err)
	assert.Len(t, fetched, 3)

	// Should be ordered by executed_at asc.
	assert.Equal(t, "/me", fetched[0].EndpointName)
	assert.Equal(t, "/drive", fetched[1].EndpointName)
	assert.Equal(t, "/mail", fetched[2].EndpointName)

	// Verify fields.
	assert.True(t, fetched[0].Success)
	assert.Equal(t, 200, fetched[0].HTTPStatus)
	assert.Equal(t, "User.Read", fetched[0].Scope)

	assert.False(t, fetched[2].Success)
	assert.Equal(t, 403, fetched[2].HTTPStatus)
	assert.Equal(t, "Insufficient privileges", fetched[2].ErrorMessage)
	assert.Equal(t, "Mail.Read", fetched[2].Scope)
}

func TestEndpointLogRepo_ListByTaskLogID_Empty(t *testing.T) {
	ctx := setupTestDB(t)

	fetched, err := EndpointLogs.ListByTaskLogID(ctx, 9999)
	require.NoError(t, err)
	assert.Empty(t, fetched)
}

func TestEndpointLogRepo_ListByTaskLogID_MultipleTaskLogs(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ep-multi", models.AuthTypeClientCredentials)

	now := time.Now()

	tl1 := &models.TaskLog{AccountID: acc.ID, RunID: "multi-1", TriggerType: models.TriggerScheduled, TotalEndpoints: 1, SuccessCount: 1, StartedAt: now}
	ep1 := []models.EndpointLog{{EndpointName: "/me", Scope: "User.Read", HTTPStatus: 200, Success: true, ExecutedAt: now}}
	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, tl1, ep1))

	tl2 := &models.TaskLog{AccountID: acc.ID, RunID: "multi-2", TriggerType: models.TriggerManual, TotalEndpoints: 2, SuccessCount: 2, StartedAt: now}
	ep2 := []models.EndpointLog{
		{EndpointName: "/users", Scope: "User.ReadAll", HTTPStatus: 200, Success: true, ExecutedAt: now},
		{EndpointName: "/groups", Scope: "Group.Read", HTTPStatus: 200, Success: true, ExecutedAt: now.Add(time.Second)},
	}
	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, tl2, ep2))

	// Each task log should only return its own endpoint logs.
	result1, err := EndpointLogs.ListByTaskLogID(ctx, tl1.ID)
	require.NoError(t, err)
	assert.Len(t, result1, 1)

	result2, err := EndpointLogs.ListByTaskLogID(ctx, tl2.ID)
	require.NoError(t, err)
	assert.Len(t, result2, 2)
}

func TestEndpointLogRepo_ResponseBodyStored(t *testing.T) {
	ctx := setupTestDB(t)
	acc := createTestAccount(t, ctx, "ep-body", models.AuthTypeClientCredentials)

	now := time.Now()
	taskLog := &models.TaskLog{
		AccountID: acc.ID, RunID: "body-run", TriggerType: models.TriggerScheduled,
		TotalEndpoints: 1, FailCount: 1, StartedAt: now,
	}
	eps := []models.EndpointLog{
		{
			EndpointName: "/mail",
			Scope:        "Mail.Read",
			HTTPStatus:   500,
			Success:      false,
			ErrorMessage: "server error",
			ResponseBody: `{"error":{"code":"InternalServerError"}}`,
			ExecutedAt:   now,
		},
	}
	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, taskLog, eps))

	fetched, err := EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
	require.NoError(t, err)
	require.Len(t, fetched, 1)
	assert.Equal(t, `{"error":{"code":"InternalServerError"}}`, fetched[0].ResponseBody)
	assert.Equal(t, "Mail.Read", fetched[0].Scope)
}
