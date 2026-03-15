package database

import (
	"context"
	"testing"
	"time"

	"e5-renewal/backend/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createTestAccount(t *testing.T, ctx context.Context, name, authType string) *models.Account {
	t.Helper()
	acc := &models.Account{
		Name:     name,
		AuthType: authType,
		AuthInfo: `{"client_id":"cid","client_secret":"csec","tenant_id":"tid"}`,
	}
	require.NoError(t, Accounts.Create(ctx, acc))
	require.NotZero(t, acc.ID)
	return acc
}

func TestAccountRepo_Create_And_GetByID(t *testing.T) {
	ctx := setupTestDB(t)

	acc := createTestAccount(t, ctx, "test-acc", models.AuthTypeClientCredentials)

	fetched, err := Accounts.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "test-acc", fetched.Name)
	assert.Equal(t, models.AuthTypeClientCredentials, fetched.AuthType)
	// AuthInfo should be decrypted back to plaintext.
	assert.Contains(t, fetched.AuthInfo, "client_id")
}

func TestAccountRepo_Create_PreservesPlaintext(t *testing.T) {
	ctx := setupTestDB(t)

	original := `{"client_id":"x","client_secret":"y","tenant_id":"z"}`
	acc := &models.Account{
		Name:     "preserve-test",
		AuthType: models.AuthTypeClientCredentials,
		AuthInfo: original,
	}
	require.NoError(t, Accounts.Create(ctx, acc))
	// After Create, caller should still see plaintext.
	assert.Equal(t, original, acc.AuthInfo)
}

func TestAccountRepo_List(t *testing.T) {
	ctx := setupTestDB(t)

	createTestAccount(t, ctx, "acc-1", models.AuthTypeAuthCode)
	createTestAccount(t, ctx, "acc-2", models.AuthTypeClientCredentials)

	accounts, err := Accounts.List(ctx)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	// Should be ordered by ID asc.
	assert.Equal(t, "acc-1", accounts[0].Name)
	assert.Equal(t, "acc-2", accounts[1].Name)
	// AuthInfo should be decrypted.
	assert.Contains(t, accounts[0].AuthInfo, "client_id")
}

func TestAccountRepo_List_Empty(t *testing.T) {
	ctx := setupTestDB(t)

	accounts, err := Accounts.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestAccountRepo_Save(t *testing.T) {
	ctx := setupTestDB(t)

	acc := createTestAccount(t, ctx, "to-update", models.AuthTypeClientCredentials)
	acc.Name = "updated-name"
	acc.AuthInfo = `{"client_id":"new","client_secret":"new","tenant_id":"new"}`
	require.NoError(t, Accounts.Save(ctx, acc))

	// After Save, caller should see plaintext.
	assert.Contains(t, acc.AuthInfo, `"client_id":"new"`)

	fetched, err := Accounts.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated-name", fetched.Name)
	assert.Contains(t, fetched.AuthInfo, `"client_id":"new"`)
}

func TestAccountRepo_GetByID_NotFound(t *testing.T) {
	ctx := setupTestDB(t)

	_, err := Accounts.GetByID(ctx, 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestAccountRepo_DeleteCascade(t *testing.T) {
	ctx := setupTestDB(t)

	acc := createTestAccount(t, ctx, "to-delete", models.AuthTypeAuthCode)

	// Create schedule, task log, and endpoint log for this account.
	sched := &models.Schedule{AccountID: acc.ID, Enabled: true}
	require.NoError(t, Schedules.Create(ctx, sched))

	taskLog := &models.TaskLog{
		AccountID:      acc.ID,
		RunID:          "run-del-1",
		TriggerType:    models.TriggerScheduled,
		TotalEndpoints: 1,
		SuccessCount:   1,
		StartedAt:      time.Now(),
	}
	epLogs := []models.EndpointLog{
		{EndpointName: "/me", HTTPStatus: 200, Success: true, ExecutedAt: time.Now()},
	}
	require.NoError(t, TaskLogs.CreateWithEndpoints(ctx, taskLog, epLogs))

	require.NoError(t, Accounts.DeleteCascade(ctx, acc.ID))

	// Account should be gone.
	_, err := Accounts.GetByID(ctx, acc.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Schedule should be gone.
	_, err = Schedules.GetByAccountID(ctx, acc.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Task logs should be gone.
	count, err := TaskLogs.CountByAccount(ctx, acc.ID)
	require.NoError(t, err)
	assert.Zero(t, count)

	// Endpoint logs should be gone.
	epLogsResult, err := EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
	require.NoError(t, err)
	assert.Empty(t, epLogsResult)
}

func TestAccountRepo_CountAll(t *testing.T) {
	ctx := setupTestDB(t)

	count, err := Accounts.CountAll(ctx)
	require.NoError(t, err)
	assert.Zero(t, count)

	createTestAccount(t, ctx, "c1", models.AuthTypeAuthCode)
	createTestAccount(t, ctx, "c2", models.AuthTypeClientCredentials)

	count, err = Accounts.CountAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestAccountRepo_CountByAuthType(t *testing.T) {
	ctx := setupTestDB(t)

	createTestAccount(t, ctx, "a1", models.AuthTypeAuthCode)
	createTestAccount(t, ctx, "a2", models.AuthTypeAuthCode)
	createTestAccount(t, ctx, "a3", models.AuthTypeClientCredentials)

	count, err := Accounts.CountByAuthType(ctx, models.AuthTypeAuthCode)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	count, err = Accounts.CountByAuthType(ctx, models.AuthTypeClientCredentials)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestAccountRepo_UpdateAuthInfo(t *testing.T) {
	ctx := setupTestDB(t)

	acc := createTestAccount(t, ctx, "auth-update", models.AuthTypeAuthCode)

	expires := time.Now().Add(24 * time.Hour)
	newInfo := `{"client_id":"new","client_secret":"new","tenant_id":"new","refresh_token":"rt"}`
	require.NoError(t, Accounts.UpdateAuthInfo(ctx, acc.ID, newInfo, &expires))

	fetched, err := Accounts.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Contains(t, fetched.AuthInfo, "refresh_token")
	assert.NotNil(t, fetched.AuthExpiresAt)
}

func TestAccountRepo_FindExpiringBefore(t *testing.T) {
	ctx := setupTestDB(t)

	// Account with expiry in the past.
	acc1 := createTestAccount(t, ctx, "expiring", models.AuthTypeAuthCode)
	past := time.Now().Add(-1 * time.Hour)
	require.NoError(t, Accounts.UpdateAuthInfo(ctx, acc1.ID, acc1.AuthInfo, &past))

	// Account with expiry in the future.
	acc2 := createTestAccount(t, ctx, "not-expiring", models.AuthTypeAuthCode)
	future := time.Now().Add(30 * 24 * time.Hour)
	require.NoError(t, Accounts.UpdateAuthInfo(ctx, acc2.ID, acc2.AuthInfo, &future))

	// Account with no expiry.
	createTestAccount(t, ctx, "no-expiry", models.AuthTypeClientCredentials)

	threshold := time.Now()
	accounts, err := Accounts.FindExpiringBefore(ctx, threshold)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "expiring", accounts[0].Name)
	// AuthInfo should be decrypted.
	assert.Contains(t, accounts[0].AuthInfo, "client_id")
}
