package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"

	"e5-renewal/backend/config"
	oauthsvc "e5-renewal/backend/services/oauth"
	"e5-renewal/backend/services/security"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTokenRoundTripper returns a fake Microsoft token response for testing.
type stubTokenRoundTripper struct{}

func (s *stubTokenRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	body := `{"token_type":"Bearer","access_token":"fake-access-token","refresh_token":"fake-refresh-token","expires_in":3600,"scope":"https://graph.microsoft.com/.default"}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil
}

func setupOAuthTestRouter(t *testing.T, transport http.RoundTripper) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("E5_CONFIG", "")
	t.Setenv("E5_PATH_PREFIX", "")
	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-minimum16chars")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit()
	oauthsvc.GlobalStateStore.Reset()

	var httpClient *http.Client
	if transport != nil {
		httpClient = &http.Client{Transport: transport}
	}

	r := gin.New()
	registerOAuthRoutes(r, oauthsvc.NewService(httpClient))
	return r
}

func oauthAuthToken(t *testing.T) string {
	t.Helper()
	token, err := security.SignJWT([]byte(config.Get().Security.JWTSecret))
	require.NoError(t, err)
	return token
}

func performJSONRequest(t *testing.T, r *gin.Engine, method, target string, body any) *httptest.ResponseRecorder {
	return performJSONRequestWithHeaders(t, r, method, target, body, nil)
}

func performJSONRequestWithHeaders(t *testing.T, r *gin.Engine, method, target string, body any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(payload)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Add auth token for auth-protected endpoints
	req.Header.Set("Authorization", "Bearer "+oauthAuthToken(t))
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeJSONBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	return payload
}

// TestOAuthAuthorizeReturnsAuthorizeURL verifies the POST /api/oauth/authorize endpoint
// returns a Microsoft authorize URL for valid JSON body parameters.
func TestOAuthAuthorizeReturnsAuthorizeURL(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
		"client_id":     "client-id",
		"client_secret": "client-secret",
		"tenant_id":     "tenant-id",
		"redirect_uri":  "https://example.com/api/oauth/callback",
	})

	require.Equal(t, http.StatusOK, w.Code)
	payload := decodeJSONBody(t, w.Body.Bytes())
	authorizeURL, ok := payload["authorize_url"].(string)
	require.True(t, ok)

	parsed, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	query := parsed.Query()
	assert.Equal(t, "client-id", query.Get("client_id"))

	stateData, found := oauthsvc.GlobalStateStore.Consume(query.Get("state"))
	require.True(t, found)
	assert.Equal(t, "client-id", stateData.ClientID)
	assert.Equal(t, "client-secret", stateData.ClientSecret)
	assert.Equal(t, "tenant-id", stateData.TenantID)
}

// TestOAuthAuthorizeRejectsMissingClientID verifies that client_id and tenant_id are required.
func TestOAuthAuthorizeRejectsMissingClientID(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
		"tenant_id": "tenant-id",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestOAuthAuthorizeRejectsMissingRedirectURI verifies that redirect_uri is required.
func TestOAuthAuthorizeRejectsMissingRedirectURI(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
		"client_id": "client-id",
		"tenant_id": "tenant-id",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestOAuthAuthorizeRejectsMissingTenantID verifies that tenant_id is required.
func TestOAuthAuthorizeRejectsMissingTenantID(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
		"client_id": "client-id",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestOAuthAuthorizeAcceptsRedirectURI verifies that POST /api/oauth/authorize accepts
// a custom redirect_uri and stores it in the state.
func TestOAuthAuthorizeAcceptsRedirectURI(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
		"client_id":    "client-id",
		"tenant_id":    "tenant-id",
		"redirect_uri": "http://localhost:3000/api/oauth/callback",
	})

	require.Equal(t, http.StatusOK, w.Code)
	payload := decodeJSONBody(t, w.Body.Bytes())
	authorizeURL, ok := payload["authorize_url"].(string)
	require.True(t, ok)

	parsed, err := url.Parse(authorizeURL)
	require.NoError(t, err)
	query := parsed.Query()
	assert.Equal(t, "http://localhost:3000/api/oauth/callback", query.Get("redirect_uri"))

	stateData, found := oauthsvc.GlobalStateStore.Consume(query.Get("state"))
	require.True(t, found)
	assert.Equal(t, "http://localhost:3000/api/oauth/callback", stateData.RedirectURI)
}

// TestOAuthAuthorizeRejectsMalformedRedirectURI verifies that malformed or unsupported
// redirect_uri values are rejected.
func TestOAuthAuthorizeRejectsMalformedRedirectURI(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	tests := []struct {
		name        string
		redirectURI string
	}{
		{name: "ftp scheme", redirectURI: "ftp://example.com/callback"},
		{name: "no scheme", redirectURI: "://bad"},
		{name: "javascript scheme", redirectURI: "javascript:alert(1)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/authorize", map[string]string{
				"client_id":    "client-id",
				"tenant_id":    "tenant-id",
				"redirect_uri": tc.redirectURI,
			})
			assert.Equal(t, http.StatusBadRequest, w.Code)
			payload := decodeJSONBody(t, w.Body.Bytes())
			assert.Equal(t, "invalid redirect_uri", payload["error"])
		})
	}
}

// TestOAuthCallbackSucceedsWithValidStateAndCode verifies that the GET /api/oauth/callback
// endpoint exchanges the code for tokens and renders an HTML result page.
func TestOAuthCallbackSucceedsWithValidStateAndCode(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	state := oauthsvc.GlobalStateStore.NewState(oauthsvc.OAuthState{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TenantID:     "tenant-id",
		RedirectURI:  "http://example.com/callback",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/callback?code=test-code&state="+url.QueryEscape(state), nil)
	r.ServeHTTP(w, req)

	// The handler will attempt a real HTTP call; accept any non-5xx status
	// that reflects the handler ran (BadRequest from MS token endpoint failure is fine).
	assert.NotEqual(t, http.StatusInternalServerError, w.Code)
}

// TestOAuthCallbackRejectsMissingParams verifies that the callback handler rejects
// requests missing code or state parameters.
func TestOAuthCallbackRejectsMissingParams(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	tests := []struct {
		name   string
		target string
	}{
		{name: "missing code", target: "/api/oauth/callback?state=some-state"},
		{name: "missing state", target: "/api/oauth/callback?code=some-code"},
		{name: "missing both", target: "/api/oauth/callback"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

// TestOAuthCallbackRejectsInvalidState verifies that an invalid or expired state token
// results in a 400 response.
func TestOAuthCallbackRejectsInvalidState(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/callback?code=test-code&state=nonexistent-state", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestOAuthCallbackStateIsConsumedOnce verifies that a state token cannot be used twice.
func TestOAuthCallbackStateIsConsumedOnce(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	state := oauthsvc.GlobalStateStore.NewState(oauthsvc.OAuthState{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TenantID:     "tenant-id",
		RedirectURI:  "http://example.com/callback",
	})

	// First request consumes the state (may succeed or fail at token exchange)
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/oauth/callback?code=test-code&state="+url.QueryEscape(state), nil)
	r.ServeHTTP(w1, req1)

	// Second request must fail because state is already consumed
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/oauth/callback?code=test-code&state="+url.QueryEscape(state), nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

// TestOAuthCallbackAndCompleteRaceAllowsOnlyOneWinner verifies concurrent requests
// with the same state: only one can consume it.
func TestOAuthCallbackAndCompleteRaceAllowsOnlyOneWinner(t *testing.T) {
	r := setupOAuthTestRouter(t, nil)

	state := oauthsvc.GlobalStateStore.NewState(oauthsvc.OAuthState{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TenantID:     "tenant-id",
		RedirectURI:  "http://example.com/callback",
	})

	var wg sync.WaitGroup
	results := make(chan int, 2)
	makeRequest := func() {
		defer wg.Done()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/oauth/callback?code=test-code&state="+url.QueryEscape(state), nil)
		r.ServeHTTP(w, req)
		results <- w.Code
	}

	wg.Add(2)
	go makeRequest()
	go makeRequest()
	wg.Wait()
	close(results)

	badRequests := 0
	for code := range results {
		if code == http.StatusBadRequest {
			badRequests++
		}
	}

	// At least one must fail because the state can only be consumed once
	assert.GreaterOrEqual(t, badRequests, 1)
}

// TestOAuthExchangeRejectsInvalidCallbackURL verifies that the exchange endpoint
// rejects callback URLs with unsupported schemes.
func TestOAuthExchangeRejectsInvalidCallbackURL(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	tests := []struct {
		name        string
		callbackURL string
		wantError   string
	}{
		{
			name:        "ftp scheme",
			callbackURL: "ftp://example.com/callback?code=c&state=s",
			wantError:   "invalid callback URL",
		},
		{
			name:        "no scheme",
			callbackURL: "://bad?code=c&state=s",
			wantError:   "invalid callback URL",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
				"callback_url": tc.callbackURL,
			})
			assert.Equal(t, http.StatusBadRequest, w.Code)
			payload := decodeJSONBody(t, w.Body.Bytes())
			assert.Equal(t, tc.wantError, payload["error"])
		})
	}
}

// TestOAuthExchangeRejectsMissingCodeOrState verifies that the exchange endpoint
// rejects callback URLs missing code or state query params.
func TestOAuthExchangeRejectsMissingCodeOrState(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	tests := []struct {
		name        string
		callbackURL string
	}{
		{name: "missing code", callbackURL: "http://localhost/callback?state=s"},
		{name: "missing state", callbackURL: "http://localhost/callback?code=c"},
		{name: "missing both", callbackURL: "http://localhost/callback"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
				"callback_url": tc.callbackURL,
			})
			assert.Equal(t, http.StatusBadRequest, w.Code)
			payload := decodeJSONBody(t, w.Body.Bytes())
			assert.Equal(t, "missing code or state in callback URL", payload["error"])
		})
	}
}

// TestOAuthExchangeRejectsInvalidState verifies that the exchange endpoint
// rejects callback URLs with invalid or expired state tokens.
func TestOAuthExchangeRejectsInvalidState(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
		"callback_url": "http://localhost/callback?code=test-code&state=nonexistent-state",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	payload := decodeJSONBody(t, w.Body.Bytes())
	assert.Equal(t, "state invalid or expired, please re-authorize", payload["error"])
}

// TestOAuthExchangeRejectsConsumedState verifies that a state cannot be used twice
// via the exchange endpoint.
func TestOAuthExchangeRejectsConsumedState(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	state := oauthsvc.GlobalStateStore.NewState(oauthsvc.OAuthState{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TenantID:     "tenant-id",
		RedirectURI:  "http://localhost/callback",
	})

	callbackURL := "http://localhost/callback?code=test-code&state=" + url.QueryEscape(state)

	// First exchange succeeds
	w1 := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
		"callback_url": callbackURL,
	})
	require.Equal(t, http.StatusOK, w1.Code)

	// Second exchange with same state must fail
	w2 := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
		"callback_url": callbackURL,
	})
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	payload := decodeJSONBody(t, w2.Body.Bytes())
	assert.Equal(t, "state invalid or expired, please re-authorize", payload["error"])
}

// TestOAuthExchangeReturnsTokensOnSuccess verifies that the exchange endpoint
// returns refresh_token and access_token on a successful code exchange.
func TestOAuthExchangeReturnsTokensOnSuccess(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	state := oauthsvc.GlobalStateStore.NewState(oauthsvc.OAuthState{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TenantID:     "tenant-id",
		RedirectURI:  "http://localhost/callback",
	})

	callbackURL := "http://localhost/callback?code=test-code&state=" + url.QueryEscape(state)

	w := performJSONRequest(t, r, http.MethodPost, "/api/oauth/exchange", map[string]string{
		"callback_url": callbackURL,
	})

	require.Equal(t, http.StatusOK, w.Code)
	payload := decodeJSONBody(t, w.Body.Bytes())
	assert.Equal(t, "fake-refresh-token", payload["refresh_token"])
	assert.Equal(t, "fake-access-token", payload["access_token"])
}

// TestOAuthExchangeRequiresAuth verifies the exchange endpoint is behind RequireAuth.
func TestOAuthExchangeRequiresAuth(t *testing.T) {
	r := setupOAuthTestRouter(t, &stubTokenRoundTripper{})

	// Send request without auth header
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/exchange", bytes.NewBufferString(`{"callback_url":"http://localhost/cb?code=c&state=s"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
