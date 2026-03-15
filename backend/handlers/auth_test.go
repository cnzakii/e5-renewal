package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/services/login"
)

func setupTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Use a unique shared-cache in-memory DB per test so all GORM connections
	// (including background goroutines) share the same database instance.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	require.NoError(t, database.Init(dsn))
	database.MustInitEncryption("test-encryption-key-1234")

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-1234")
	config.MustInit()
	login.MustInit("test-login-key")

	r := gin.New()
	handlers.RegisterAuthRoutes(r)
	return r
}

func TestLoginSuccess(t *testing.T) {
	r := setupTestEngine(t)

	body, _ := json.Marshal(map[string]string{"key": "test-login-key"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
}

func TestLoginWrongKey(t *testing.T) {
	r := setupTestEngine(t)

	body, _ := json.Marshal(map[string]string{"key": "wrong-key"})
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginRateLimit(t *testing.T) {
	r := setupTestEngine(t)

	for i := 0; i < 6; i++ {
		body, _ := json.Marshal(map[string]string{"key": "wrong"})
		req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if i < 5 {
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}
