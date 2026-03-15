package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"e5-renewal/backend/middleware"
)

func setupRateLimitEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", middleware.LoginRateLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func doLoginRequest(r *gin.Engine, ip string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = ip + ":12345"
	r.ServeHTTP(w, req)
	return w
}

func TestLoginRateLimit_UnderLimit(t *testing.T) {
	r := setupRateLimitEngine()

	// First 5 requests should all succeed
	for i := 0; i < 5; i++ {
		w := doLoginRequest(r, "10.0.0.1")
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed", i+1)
	}
}

func TestLoginRateLimit_AtLimit(t *testing.T) {
	r := setupRateLimitEngine()

	// Exhaust the 5-request limit
	for i := 0; i < 5; i++ {
		w := doLoginRequest(r, "10.0.0.2")
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 6th request should be rate-limited
	w := doLoginRequest(r, "10.0.0.2")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "too many login attempts")
}

func TestLoginRateLimit_MultipleOverLimit(t *testing.T) {
	r := setupRateLimitEngine()

	// Exhaust the limit
	for i := 0; i < 5; i++ {
		doLoginRequest(r, "10.0.0.3")
	}

	// Several subsequent requests should all be blocked
	for i := 0; i < 3; i++ {
		w := doLoginRequest(r, "10.0.0.3")
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "over-limit request %d should be blocked", i+1)
	}
}

func TestLoginRateLimit_DifferentIPs(t *testing.T) {
	r := setupRateLimitEngine()

	// Exhaust limit for IP A
	for i := 0; i < 5; i++ {
		doLoginRequest(r, "192.168.1.1")
	}

	// IP A is now blocked
	w := doLoginRequest(r, "192.168.1.1")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// IP B should still be allowed
	w = doLoginRequest(r, "192.168.1.2")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoginRateLimit_DifferentIPsIndependent(t *testing.T) {
	r := setupRateLimitEngine()

	// Use 3 requests from IP A
	for i := 0; i < 3; i++ {
		w := doLoginRequest(r, "172.16.0.1")
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// Use 5 requests from IP B (exhaust it)
	for i := 0; i < 5; i++ {
		w := doLoginRequest(r, "172.16.0.2")
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// IP B is blocked
	w := doLoginRequest(r, "172.16.0.2")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// IP A still has 2 requests left
	w = doLoginRequest(r, "172.16.0.1")
	assert.Equal(t, http.StatusOK, w.Code)

	w = doLoginRequest(r, "172.16.0.1")
	assert.Equal(t, http.StatusOK, w.Code)

	// Now IP A is also exhausted
	w = doLoginRequest(r, "172.16.0.1")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
