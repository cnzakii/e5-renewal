package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"e5-renewal/backend/config"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/services/security"
)

func initConfig(t *testing.T) {
	t.Helper()
	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-minimum16chars")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit()
}

func setupAuthEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	initConfig(t)

	r := gin.New()
	r.GET("/protected", middleware.RequireAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	r := setupAuthEngine(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing bearer token")
}

func TestRequireAuth_EmptyBearerToken(t *testing.T) {
	r := setupAuthEngine(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)

	// Empty string after "Bearer " is an invalid JWT
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token")
}

func TestRequireAuth_NoBearerPrefix(t *testing.T) {
	r := setupAuthEngine(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic some-credentials")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing bearer token")
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	r := setupAuthEngine(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt-token")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token")
}

func TestRequireAuth_WrongSecret(t *testing.T) {
	r := setupAuthEngine(t)

	// Sign with a different secret
	token, err := security.SignJWT([]byte("different-secret-minimum16chars!"))
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid token")
}

func TestRequireAuth_ValidToken(t *testing.T) {
	r := setupAuthEngine(t)

	secret := []byte(config.Get().Security.JWTSecret)
	token, err := security.SignJWT(secret)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}
