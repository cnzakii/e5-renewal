package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/services/security"
)

func TestHealthBasic(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterHealthRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "connected", resp["db"])
}

func TestHealthDetailRequiresAuth(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterHealthRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/health/detail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHealthDetailWithAuth(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterHealthRoutes(r)

	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/health/detail", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ok", resp["status"])
	assert.Equal(t, "connected", resp["db"])
	assert.NotNil(t, resp["uptime_seconds"])
	assert.Equal(t, float64(0), resp["accounts_count"])
	assert.Nil(t, resp["last_run_at"])
}
