package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/handlers"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/security"
)

func settingEngine(t *testing.T) *settingTestHelper {
	t.Helper()
	r := setupTestEngine(t)
	handlers.RegisterSettingRoutes(r)
	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)
	return &settingTestHelper{r: r, token: token}
}

type settingTestHelper struct {
	r interface {
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
	token string
}

func (h *settingTestHelper) do(t *testing.T, method, url string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	req.Header.Set("Authorization", "Bearer "+h.token)
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)
	return w
}

func TestGetNotificationSettingsDefault(t *testing.T) {
	h := settingEngine(t)

	w := h.do(t, http.MethodGet, "/api/settings/notification", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.NotificationConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	// Default values when no setting exists
	assert.Equal(t, 7, resp.ExpiryDaysBefore)
	assert.Equal(t, 50, resp.HealthThreshold)
	assert.Empty(t, resp.URL)
}

func TestUpdateNotificationSettings(t *testing.T) {
	h := settingEngine(t)

	cfg := models.NotificationConfig{
		URL:              "https://example.com/webhook",
		OnAuthExpiry:     true,
		ExpiryDaysBefore: 14,
		OnTaskAllFailed:  true,
		OnHealthLow:      true,
		HealthThreshold:  30,
	}

	w := h.do(t, http.MethodPut, "/api/settings/notification", cfg)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the setting was saved
	w = h.do(t, http.MethodGet, "/api/settings/notification", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.NotificationConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "https://example.com/webhook", resp.URL)
	assert.True(t, resp.OnAuthExpiry)
	assert.Equal(t, 14, resp.ExpiryDaysBefore)
	assert.True(t, resp.OnTaskAllFailed)
	assert.True(t, resp.OnHealthLow)
	assert.Equal(t, 30, resp.HealthThreshold)
}

func TestUpdateNotificationSettingsInvalidBody(t *testing.T) {
	h := settingEngine(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/notification", bytes.NewReader([]byte("not json")))
	req.Header.Set("Authorization", "Bearer "+h.token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestNotificationSettingsRequiresAuth(t *testing.T) {
	r := setupTestEngine(t)
	handlers.RegisterSettingRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/api/settings/notification", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	req = httptest.NewRequest(http.MethodPut, "/api/settings/notification", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetNotificationSettingsDefaultLanguage(t *testing.T) {
	h := settingEngine(t)
	w := h.do(t, http.MethodGet, "/api/settings/notification", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp models.NotificationConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "zh", resp.Language)
}

func TestUpdateNotificationSettingsWithLanguage(t *testing.T) {
	h := settingEngine(t)
	cfg := models.NotificationConfig{
		URL:      "https://example.com/hook",
		Language: "en",
	}
	w := h.do(t, http.MethodPut, "/api/settings/notification", cfg)
	assert.Equal(t, http.StatusOK, w.Code)

	w = h.do(t, http.MethodGet, "/api/settings/notification", nil)
	var resp models.NotificationConfig
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "en", resp.Language)
}

func TestTestNotificationNoSettingSaved(t *testing.T) {
	h := settingEngine(t)

	w := h.do(t, http.MethodPost, "/api/settings/notification/test", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "not found")
}

func TestTestNotificationEmptyURL(t *testing.T) {
	h := settingEngine(t)

	// Save a config with empty URL
	cfg := models.NotificationConfig{
		URL:          "",
		OnAuthExpiry: true,
	}
	w := h.do(t, http.MethodPut, "/api/settings/notification", cfg)
	assert.Equal(t, http.StatusOK, w.Code)

	w = h.do(t, http.MethodPost, "/api/settings/notification/test", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["error"], "empty")
}
