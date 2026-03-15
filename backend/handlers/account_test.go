package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/executor"
	"e5-renewal/backend/services/oauth"
	"e5-renewal/backend/services/scheduler"
	"e5-renewal/backend/services/security"
)

func setupAccountEngine(t *testing.T) (*gin.Engine, *scheduler.Scheduler) {
	t.Helper()
	r := setupTestEngine(t)
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauth.NewService(nil), rng)
	sched := scheduler.New(exec, rand.New(rand.NewSource(42)))
	sched.Start(context.Background())
	t.Cleanup(sched.Stop)
	handlers.RegisterAccountRoutes(r, sched)
	return r, sched
}

func accountAuthToken(t *testing.T) string {
	t.Helper()
	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)
	return token
}

func doAccountReq(t *testing.T, r *gin.Engine, method, url string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	token := accountAuthToken(t)
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- maskSecret is not exported, but we can test it indirectly via list/create responses ---

func TestListAccountsEmpty(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 0)
}

func TestListAccountsRequiresAuth(t *testing.T) {
	r, _ := setupAccountEngine(t)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateAccountSuccess(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name":           "test-account",
		"auth_type":      "client_credentials",
		"client_id":      "my-client-id",
		"client_secret":  "my-client-secret-long",
		"tenant_id":      "my-tenant-id",
		"notify_enabled": true,
	}

	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotZero(t, resp["id"])
}

func TestCreateAccountWithAuthCode(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name":            "auth-code-acc",
		"auth_type":       "auth_code",
		"client_id":       "cid",
		"client_secret":   "my-client-secret-value",
		"tenant_id":       "tid",
		"refresh_token":   "my-refresh-token-value",
		"auth_expires_at": "2025-12-31",
	}

	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	assert.Equal(t, http.StatusCreated, w.Code)

	// Now list to check that secrets are masked
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)

	acc := accounts[0]
	assert.Equal(t, "auth-code-acc", acc["name"])
	assert.Equal(t, "auth_code", acc["auth_type"])
	assert.Equal(t, "cid", acc["client_id"])
	assert.Equal(t, "tid", acc["tenant_id"])
	assert.Equal(t, "2025-12-31", acc["auth_expires_at"])
	// Secrets should be masked (maskSecret: first4 + 8 stars + last4 for len>8)
	assert.Equal(t, "my-c********alue", acc["client_secret"])
	assert.Equal(t, "my-r********alue", acc["refresh_token"])
}

func TestCreateAccountInvalidAuthType(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name":      "bad-type",
		"auth_type": "invalid",
		"client_id": "cid",
	}

	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccountInvalidJSON(t *testing.T) {
	r, _ := setupAccountEngine(t)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accountAuthToken(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateAccountSuccess(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create
	createBody := map[string]interface{}{
		"name":          "original",
		"auth_type":     "client_credentials",
		"client_id":     "cid",
		"client_secret": "csecret",
		"tenant_id":     "tid",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	id := int(createResp["id"].(float64))

	// Update
	updateBody := map[string]interface{}{
		"name":          "updated",
		"auth_type":     "client_credentials",
		"client_id":     "new-cid",
		"client_secret": "new-csec",
		"tenant_id":     "new-tid",
	}
	w = doAccountReq(t, r, http.MethodPut, fmt.Sprintf("/api/accounts/%d", id), updateBody)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	assert.Equal(t, "updated", accounts[0]["name"])
	assert.Equal(t, "new-cid", accounts[0]["client_id"])
	assert.Equal(t, "new-tid", accounts[0]["tenant_id"])
}

func TestUpdateAccountNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name":      "x",
		"auth_type": "client_credentials",
		"client_id": "cid",
	}
	w := doAccountReq(t, r, http.MethodPut, "/api/accounts/9999", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateAccountInvalidID(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodPut, "/api/accounts/abc", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateAccountInvalidAuthType(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create first
	createBody := map[string]interface{}{
		"name": "acc", "auth_type": "client_credentials",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := int(resp["id"].(float64))

	updateBody := map[string]interface{}{
		"name": "acc", "auth_type": "invalid",
	}
	w = doAccountReq(t, r, http.MethodPut, fmt.Sprintf("/api/accounts/%d", id), updateBody)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteAccountSuccess(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create
	createBody := map[string]interface{}{
		"name": "to-delete", "auth_type": "client_credentials",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := int(resp["id"].(float64))

	// Delete
	w = doAccountReq(t, r, http.MethodDelete, fmt.Sprintf("/api/accounts/%d", id), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify gone
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	assert.Len(t, accounts, 0)
}

func TestDeleteAccountNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodDelete, "/api/accounts/9999", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAccountInvalidID(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodDelete, "/api/accounts/abc", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTriggerAccountNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodPost, "/api/accounts/9999/trigger", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTriggerAccountInvalidID(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodPost, "/api/accounts/abc/trigger", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetSchedule(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create account (which also creates a schedule)
	createBody := map[string]interface{}{
		"name": "sched-acc", "auth_type": "client_credentials",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := int(resp["id"].(float64))

	// Get schedule
	w = doAccountReq(t, r, http.MethodGet, fmt.Sprintf("/api/accounts/%d/schedule", id), nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var sched map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sched))
	assert.Equal(t, false, sched["enabled"])
	assert.Equal(t, false, sched["paused"])
	assert.Equal(t, float64(30), sched["pause_threshold"])
}

func TestGetScheduleNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodGet, "/api/accounts/9999/schedule", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetScheduleInvalidID(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodGet, "/api/accounts/abc/schedule", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSchedule(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create account
	createBody := map[string]interface{}{
		"name": "sched-upd", "auth_type": "client_credentials",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := int(resp["id"].(float64))

	// Update schedule: enable it
	enabled := true
	threshold := 50
	schedBody := map[string]interface{}{
		"enabled":         enabled,
		"pause_threshold": threshold,
	}
	w = doAccountReq(t, r, http.MethodPut, fmt.Sprintf("/api/accounts/%d/schedule", id), schedBody)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify
	w = doAccountReq(t, r, http.MethodGet, fmt.Sprintf("/api/accounts/%d/schedule", id), nil)
	var sched map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sched))
	assert.Equal(t, true, sched["enabled"])
	assert.Equal(t, float64(50), sched["pause_threshold"])
}

func TestUpdateScheduleNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{"enabled": true}
	w := doAccountReq(t, r, http.MethodPut, "/api/accounts/9999/schedule", body)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateScheduleInvalidID(t *testing.T) {
	r, _ := setupAccountEngine(t)

	w := doAccountReq(t, r, http.MethodPut, "/api/accounts/abc/schedule", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateScheduleUnpauseClearsReason(t *testing.T) {
	r, _ := setupAccountEngine(t)

	ctx := context.Background()
	// Create account and schedule
	acc := models.Account{Name: "pause-test", AuthType: models.AuthTypeClientCredentials,
		AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))
	sched := models.Schedule{
		AccountID: acc.ID, Enabled: true, Paused: true,
		PauseReason: "health low", PauseThreshold: 30,
	}
	require.NoError(t, database.Schedules.Create(ctx, &sched))

	// Unpause
	body := map[string]interface{}{"paused": false}
	w := doAccountReq(t, r, http.MethodPut, fmt.Sprintf("/api/accounts/%d/schedule", acc.ID), body)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify pause_reason is cleared
	w = doAccountReq(t, r, http.MethodGet, fmt.Sprintf("/api/accounts/%d/schedule", acc.ID), nil)
	var schedResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &schedResp))
	assert.Equal(t, false, schedResp["paused"])
	assert.Equal(t, "", schedResp["pause_reason"])
}

func TestListAccountsWithScheduleInfo(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create an account (creates schedule with defaults)
	createBody := map[string]interface{}{
		"name": "with-sched", "auth_type": "client_credentials",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)

	// List accounts - should include schedule info
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	sched, ok := accounts[0]["schedule"].(map[string]interface{})
	require.True(t, ok, "schedule should be present")
	assert.Equal(t, false, sched["enabled"])
	assert.Equal(t, float64(30), sched["pause_threshold"])
}

func TestMaskSecretShortString(t *testing.T) {
	// Test maskSecret indirectly: create account with short secret, check list response
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name": "short-secret", "auth_type": "client_credentials",
		"client_id": "cid", "client_secret": "abcd", "tenant_id": "tid",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	require.Equal(t, http.StatusCreated, w.Code)

	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	// Short secrets (<=8 chars) are fully masked
	assert.Equal(t, "****", accounts[0]["client_secret"])
}

func TestMaskSecretLongString(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name": "long-secret", "auth_type": "client_credentials",
		"client_id": "cid", "client_secret": "abcdefghijklmnop", "tenant_id": "tid",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	require.Equal(t, http.StatusCreated, w.Code)

	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	// Long secrets: first4 + 8 stars + last4
	assert.Equal(t, "abcd********mnop", accounts[0]["client_secret"])
}

func TestCreateAccountWithAuthExpiresAt(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"name":            "expiry-test",
		"auth_type":       "auth_code",
		"client_id":       "cid",
		"client_secret":   "csec",
		"tenant_id":       "tid",
		"refresh_token":   "rtok",
		"auth_expires_at": "2025-06-15",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	require.Equal(t, http.StatusCreated, w.Code)

	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	assert.Equal(t, "2025-06-15", accounts[0]["auth_expires_at"])
}

func TestUpdateAccountClearsAuthExpiresAt(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create with expiry
	body := map[string]interface{}{
		"name": "clear-expiry", "auth_type": "auth_code",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
		"refresh_token": "r", "auth_expires_at": "2025-06-15",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	id := int(resp["id"].(float64))

	// Update without auth_expires_at (should clear it)
	updateBody := map[string]interface{}{
		"name": "clear-expiry", "auth_type": "auth_code",
		"client_id": "c", "client_secret": "s", "tenant_id": "t",
		"refresh_token": "r",
	}
	w = doAccountReq(t, r, http.MethodPut, fmt.Sprintf("/api/accounts/%d", id), updateBody)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)
	assert.Equal(t, "", accounts[0]["auth_expires_at"])
}

func TestAccountHealthStats(t *testing.T) {
	r, _ := setupAccountEngine(t)

	ctx := context.Background()
	acc := models.Account{Name: "stats-acc", AuthType: models.AuthTypeClientCredentials,
		AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	// Create a schedule so list returns it
	require.NoError(t, database.Schedules.Create(ctx, &models.Schedule{
		AccountID: acc.ID, Enabled: false, PauseThreshold: 30,
	}))

	// Create some task logs
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		tl := models.TaskLog{
			AccountID: acc.ID, RunID: fmt.Sprintf("stats-%d", i),
			TriggerType: models.TriggerScheduled, TotalEndpoints: 10, SuccessCount: 8, FailCount: 2,
			StartedAt: now.Add(time.Duration(-i) * time.Hour),
		}
		require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))
	}

	w := doAccountReq(t, r, http.MethodGet, "/api/accounts", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var accounts []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &accounts))
	require.Len(t, accounts, 1)

	acc0 := accounts[0]
	assert.Equal(t, float64(30), acc0["total_runs"])   // 3 logs * 10 endpoints
	assert.Equal(t, float64(24), acc0["success_runs"]) // 3 logs * 8 success
	assert.NotNil(t, acc0["last_run"])
	assert.NotNil(t, acc0["health"])
	assert.Equal(t, float64(80), acc0["health"]) // 24/30 * 100 = 80%
}

func TestGetAccountByIDUnmasked(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// First create an auth_code account
	body := map[string]interface{}{
		"name":          "unmasked-test",
		"auth_type":     "auth_code",
		"client_id":     "cid-123",
		"client_secret": "my-client-secret-value",
		"tenant_id":     "tid-456",
		"refresh_token": "my-refresh-token-value",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", body)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	id := fmt.Sprintf("%.0f", createResp["id"].(float64))

	// GET /accounts/:id should return unmasked values
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts/"+id, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var acc map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acc))
	assert.Equal(t, "my-client-secret-value", acc["client_secret"])
	assert.Equal(t, "my-refresh-token-value", acc["refresh_token"])
	assert.Equal(t, "cid-123", acc["client_id"])
	assert.Equal(t, "tid-456", acc["tenant_id"])
}

func TestGetAccountByIDNotFound(t *testing.T) {
	r, _ := setupAccountEngine(t)
	w := doAccountReq(t, r, http.MethodGet, "/api/accounts/99999", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAccountByIDRequiresAuth(t *testing.T) {
	r, _ := setupAccountEngine(t)
	req := httptest.NewRequest(http.MethodGet, "/api/accounts/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateAccountMaskedSecretPreserved(t *testing.T) {
	r, _ := setupAccountEngine(t)

	// Create account
	createBody := map[string]interface{}{
		"name":          "mask-guard-test",
		"auth_type":     "auth_code",
		"client_id":     "cid",
		"client_secret": "original-secret-value",
		"tenant_id":     "tid",
		"refresh_token": "original-refresh-token",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts", createBody)
	require.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	id := fmt.Sprintf("%.0f", createResp["id"].(float64))

	// Update with masked values (simulating the frontend sending masked data)
	updateBody := map[string]interface{}{
		"name":          "mask-guard-test",
		"auth_type":     "auth_code",
		"client_id":     "cid",
		"client_secret": "orig********alue",
		"tenant_id":     "tid",
		"refresh_token": "orig********oken",
	}
	w = doAccountReq(t, r, http.MethodPut, "/api/accounts/"+id, updateBody)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify via GET /accounts/:id that real values were not overwritten
	w = doAccountReq(t, r, http.MethodGet, "/api/accounts/"+id, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var acc map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &acc))
	assert.Equal(t, "original-secret-value", acc["client_secret"])
	assert.Equal(t, "original-refresh-token", acc["refresh_token"])
}

func TestVerifyAccountInvalidAuthType(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"auth_type": "invalid",
		"client_id": "cid",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts/verify", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyAccountAuthCodeMissingRefreshToken(t *testing.T) {
	r, _ := setupAccountEngine(t)

	body := map[string]interface{}{
		"auth_type":     "auth_code",
		"client_id":     "cid",
		"client_secret": "csec",
		"tenant_id":     "tid",
	}
	w := doAccountReq(t, r, http.MethodPost, "/api/accounts/verify", body)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "refresh_token")
}

func TestVerifyAccountInvalidJSON(t *testing.T) {
	r, _ := setupAccountEngine(t)

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/verify", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accountAuthToken(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
