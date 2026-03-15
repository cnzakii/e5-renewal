package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
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

func authToken(t *testing.T) string {
	t.Helper()
	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)
	return token
}

func authedRequest(t *testing.T, method, url string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, url, nil)
	req.Header.Set("Authorization", "Bearer "+authToken(t))
	return req
}

func TestListTaskLogsRequiresAuth(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListTaskLogsEmpty(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	req := authedRequest(t, http.MethodGet, "/api/logs")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["total"])
	assert.Equal(t, float64(1), resp["page"])
	assert.Equal(t, float64(20), resp["page_size"])
}

func TestListTaskLogsWithData(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	ctx := context.Background()
	// Create an account first
	acc := models.Account{
		Name:     "test-acc",
		AuthType: models.AuthTypeClientCredentials,
		AuthInfo: `{"client_id":"cid","client_secret":"csec","tenant_id":"tid"}`,
	}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	now := time.Now().UTC()
	finished := now.Add(time.Minute)
	taskLog := models.TaskLog{
		AccountID:      acc.ID,
		RunID:          "run-1",
		TriggerType:    models.TriggerManual,
		TotalEndpoints: 5,
		SuccessCount:   4,
		FailCount:      1,
		StartedAt:      now,
		FinishedAt:     &finished,
	}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &taskLog, nil))

	req := authedRequest(t, http.MethodGet, "/api/logs")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
	items := resp["items"].([]interface{})
	require.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "test-acc", item["account_name"])
	assert.Equal(t, "manual", item["trigger_type"])
	assert.Equal(t, float64(5), item["total_endpoints"])
	assert.Equal(t, float64(4), item["success_count"])
	assert.Equal(t, float64(1), item["fail_count"])
}

func TestListTaskLogsPagination(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	ctx := context.Background()
	acc := models.Account{
		Name:     "test-acc",
		AuthType: models.AuthTypeClientCredentials,
		AuthInfo: `{"client_id":"cid","client_secret":"csec","tenant_id":"tid"}`,
	}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	// Create 3 logs
	for i := 0; i < 3; i++ {
		tl := models.TaskLog{
			AccountID:      acc.ID,
			RunID:          fmt.Sprintf("run-%d", i),
			TriggerType:    models.TriggerScheduled,
			TotalEndpoints: 1,
			SuccessCount:   1,
			StartedAt:      time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, nil))
	}

	// Page 1, size 2
	req := authedRequest(t, http.MethodGet, "/api/logs?page=1&page_size=2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp["total"])
	assert.Equal(t, float64(2), resp["page_size"])
	items := resp["items"].([]interface{})
	assert.Len(t, items, 2)

	// Page 2
	req = authedRequest(t, http.MethodGet, "/api/logs?page=2&page_size=2")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	items = resp["items"].([]interface{})
	assert.Len(t, items, 1)
}

func TestListTaskLogsMaxPageSize(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	req := authedRequest(t, http.MethodGet, "/api/logs?page_size=999")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// maxPageSize is 100
	assert.Equal(t, float64(100), resp["page_size"])
}

func TestListTaskLogsFilterByAccountID(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	ctx := context.Background()
	acc1 := models.Account{Name: "acc1", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c1","client_secret":"s1","tenant_id":"t1"}`}
	acc2 := models.Account{Name: "acc2", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c2","client_secret":"s2","tenant_id":"t2"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc1))
	require.NoError(t, database.Accounts.Create(ctx, &acc2))

	tl1 := models.TaskLog{AccountID: acc1.ID, RunID: "r1", TriggerType: models.TriggerManual, TotalEndpoints: 1, SuccessCount: 1, StartedAt: time.Now().UTC()}
	tl2 := models.TaskLog{AccountID: acc2.ID, RunID: "r2", TriggerType: models.TriggerManual, TotalEndpoints: 1, SuccessCount: 1, StartedAt: time.Now().UTC()}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl1, nil))
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl2, nil))

	req := authedRequest(t, http.MethodGet, fmt.Sprintf("/api/logs?account_id=%d", acc1.ID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
}

func TestListEndpointLogs(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	ctx := context.Background()
	acc := models.Account{Name: "test", AuthType: models.AuthTypeClientCredentials, AuthInfo: `{"client_id":"c","client_secret":"s","tenant_id":"t"}`}
	require.NoError(t, database.Accounts.Create(ctx, &acc))

	now := time.Now().UTC()
	tl := models.TaskLog{AccountID: acc.ID, RunID: "run-ep", TriggerType: models.TriggerManual, TotalEndpoints: 1, SuccessCount: 1, StartedAt: now}
	eps := []models.EndpointLog{{EndpointName: "GET /me", Scope: "User.Read", HTTPStatus: 200, Success: true, ExecutedAt: now}}
	require.NoError(t, database.TaskLogs.CreateWithEndpoints(ctx, &tl, eps))

	req := authedRequest(t, http.MethodGet, fmt.Sprintf("/api/logs/%d/endpoints", tl.ID))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "GET /me", result[0]["endpoint_name"])
	assert.Equal(t, "User.Read", result[0]["scope"])
}

func TestListEndpointLogsInvalidID(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterLogRoutes(r)

	req := authedRequest(t, http.MethodGet, "/api/logs/abc/endpoints")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
