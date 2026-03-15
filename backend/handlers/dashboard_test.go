package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/security"
)

func dashboardEngine(t *testing.T) (*httptest.ResponseRecorder, func(method, url string) *httptest.ResponseRecorder) {
	t.Helper()
	r := setupTestEngine(t)
	handlers.RegisterDashboardRoutes(r)
	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)

	do := func(method, url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	return nil, do
}

func TestDashboardSummaryRequiresAuth(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterDashboardRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDashboardSummaryEmpty(t *testing.T) {
	_, do := dashboardEngine(t)

	w := do(http.MethodGet, "/api/dashboard/summary")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total_accounts"])
	assert.Equal(t, float64(0), resp["total_runs"])
	assert.Equal(t, float64(0), resp["error_count"])
	assert.Equal(t, float64(0), resp["success_rate"])
}

func TestDashboardSummaryWithData(t *testing.T) {
	_, do := dashboardEngine(t)

	ctx := context.Background()
	// Create accounts of different types
	acc1 := models.Account{Name: "auth-code-acc", AuthType: models.AuthTypeAuthCode, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t","refresh_token":"r"}`}
	acc2 := models.Account{Name: "cred-acc", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc1))
	require.NoError(t, database.Accounts.Create(ctx, &acc2))

	// Create task logs
	now := time.Now().UTC()
	tl := models.TaskLog{AccountID: acc1.ID, RunID: "run-d1", TriggerType: models.TriggerScheduled, TotalEndpoints: 5, SuccessCount: 4, FailCount: 1, StartedAt: now}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))

	w := do(http.MethodGet, "/api/dashboard/summary")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["total_accounts"])
	assert.Equal(t, float64(1), resp["auth_code_count"])
	assert.Equal(t, float64(1), resp["credentials_count"])
	// total_runs counts task log rows (1 log created), not individual endpoints
	assert.Equal(t, float64(1), resp["total_runs"])
	// error_count = logs with fail_count > 0 (this log has fail_count=1)
	assert.Equal(t, float64(1), resp["error_count"])
	// success_rate = (total - errors) / total * 100 = 0/1 * 100 = 0
	assert.Equal(t, float64(0), resp["success_rate"])
}

func TestDashboardSummaryPeriodFilter(t *testing.T) {
	_, do := dashboardEngine(t)

	ctx := context.Background()
	acc := models.Account{Name: "acc", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	// Create a log from 10 days ago
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	tl := models.TaskLog{AccountID: acc.ID, RunID: "old-run", TriggerType: models.TriggerScheduled, TotalEndpoints: 3, SuccessCount: 3, StartedAt: oldTime}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))

	// 1d should show 0 runs
	w := do(http.MethodGet, "/api/dashboard/summary?period=1d")
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total_runs"])

	// 7d should show 0 runs (10 days ago)
	w = do(http.MethodGet, "/api/dashboard/summary?period=7d")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total_runs"])

	// 30d should show 1 task log run (TotalEndpoints=3 but total_runs counts log rows)
	w = do(http.MethodGet, "/api/dashboard/summary?period=30d")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total_runs"])
}

func TestDashboardTrend(t *testing.T) {
	_, do := dashboardEngine(t)

	w := do(http.MethodGet, "/api/dashboard/trend")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Default 7d gives 7 buckets
	assert.Len(t, resp, 7)
}

func TestDashboardTrend1d(t *testing.T) {
	_, do := dashboardEngine(t)

	w := do(http.MethodGet, "/api/dashboard/trend?period=1d")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// 1d period gives 24 hourly buckets
	assert.Len(t, resp, 24)
}

func TestDashboardAccountHealth(t *testing.T) {
	_, do := dashboardEngine(t)

	ctx := context.Background()
	acc := models.Account{Name: "health-acc", AuthType: models.AuthTypeAuthCode, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t","refresh_token":"r"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	w := do(http.MethodGet, "/api/dashboard/account-health")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "health-acc", resp[0]["name"])
	assert.Equal(t, "auth_code", resp[0]["auth_type"])
}

func TestDashboardRecentLogs(t *testing.T) {
	_, do := dashboardEngine(t)

	ctx := context.Background()
	acc := models.Account{Name: "recent-acc", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	now := time.Now().UTC()
	finished := now.Add(time.Second)
	tl := models.TaskLog{AccountID: acc.ID, RunID: "recent-1", TriggerType: models.TriggerManual, TotalEndpoints: 2, SuccessCount: 1, FailCount: 1, StartedAt: now, FinishedAt: &finished}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))

	w := do(http.MethodGet, "/api/dashboard/recent-logs")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp, 1)
	assert.Equal(t, "recent-acc", resp[0]["account_name"])
	assert.Equal(t, "manual", resp[0]["trigger_type"])
	assert.Equal(t, float64(2), resp[0]["total_endpoints"])
	assert.Equal(t, float64(1), resp[0]["success_count"])
	assert.Equal(t, float64(1), resp[0]["fail_count"])
	assert.NotEmpty(t, resp[0]["started_at"])
	assert.NotEmpty(t, resp[0]["finished_at"])
}
