package oauth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/oauth"
)

func TestNewService_NilClient(t *testing.T) {
	svc := oauth.NewService(nil)
	assert.NotNil(t, svc)
}

func TestNewService_CustomClient(t *testing.T) {
	client := &http.Client{}
	svc := oauth.NewService(client)
	assert.NotNil(t, svc)
}

func TestBuildAuthorizeURL_WithTenantID(t *testing.T) {
	svc := oauth.NewService(nil)
	url := svc.BuildAuthorizeURL("my-tenant", "my-client", "http://localhost/callback", "abc123", []string{"User.Read", "Mail.Read"})

	assert.Contains(t, url, "https://login.microsoftonline.com/my-tenant/oauth2/v2.0/authorize")
	assert.Contains(t, url, "client_id=my-client")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "redirect_uri=")
	assert.Contains(t, url, "state=abc123")
	assert.Contains(t, url, "scope=User.Read+Mail.Read")
}

func TestBuildAuthorizeURL_EmptyTenantDefaultsToCommon(t *testing.T) {
	svc := oauth.NewService(nil)
	url := svc.BuildAuthorizeURL("", "my-client", "http://localhost/callback", "state1", []string{"User.Read"})

	assert.Contains(t, url, "https://login.microsoftonline.com/common/oauth2/v2.0/authorize")
}

func TestRefreshToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "test-client", r.FormValue("client_id"))
		assert.Equal(t, "test-secret", r.FormValue("client_secret"))
		assert.Equal(t, "refresh_token", r.FormValue("grant_type"))
		assert.Equal(t, "old-refresh-token", r.FormValue("refresh_token"))

		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create a custom transport that redirects Microsoft URLs to our test server
	svc := oauth.NewService(newRedirectClient(server.URL))

	resp, err := svc.RefreshToken(context.Background(), "test-tenant", "test-client", "test-secret", "old-refresh-token", "https://graph.microsoft.com/User.Read offline_access")
	require.NoError(t, err)
	assert.Equal(t, "new-access-token", resp.AccessToken)
	assert.Equal(t, "new-refresh-token", resp.RefreshToken)
}

func TestAcquireClientToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "client_credentials", r.FormValue("grant_type"))

		resp := oauth.TokenResponse{
			TokenType:   "Bearer",
			AccessToken: "client-access-token",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := oauth.NewService(newRedirectClient(server.URL))

	resp, err := svc.AcquireClientToken(context.Background(), "test-tenant", "test-client", "test-secret")
	require.NoError(t, err)
	assert.Equal(t, "client-access-token", resp.AccessToken)
}

func TestExchangeAuthCode_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.Equal(t, "auth-code-123", r.FormValue("code"))
		assert.Equal(t, "http://localhost/callback", r.FormValue("redirect_uri"))

		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "exchanged-access-token",
			RefreshToken: "exchanged-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := oauth.NewService(newRedirectClient(server.URL))

	resp, err := svc.ExchangeAuthCode(context.Background(), "test-tenant", "test-client", "test-secret", "auth-code-123", "http://localhost/callback", "https://graph.microsoft.com/User.Read offline_access")
	require.NoError(t, err)
	assert.Equal(t, "exchanged-access-token", resp.AccessToken)
	assert.Equal(t, "exchanged-refresh-token", resp.RefreshToken)
}

func TestRefreshToken_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(oauth.ErrorResponse{
			Error:       "invalid_grant",
			Description: "The refresh token is expired",
		})
	}))
	defer server.Close()

	svc := oauth.NewService(newRedirectClient(server.URL))

	_, err := svc.RefreshToken(context.Background(), "test-tenant", "test-client", "test-secret", "bad-token", "https://graph.microsoft.com/User.Read offline_access")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_grant")
	assert.Contains(t, err.Error(), "expired")
}

func TestRefreshToken_NetworkError(t *testing.T) {
	// Use a server that's already closed to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	svc := oauth.NewService(newRedirectClient(serverURL))

	_, err := svc.RefreshToken(context.Background(), "test-tenant", "test-client", "test-secret", "token", "https://graph.microsoft.com/User.Read offline_access")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token request")
}

func TestRefreshToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	svc := oauth.NewService(newRedirectClient(server.URL))

	_, err := svc.RefreshToken(context.Background(), "test-tenant", "test-client", "test-secret", "token", "https://graph.microsoft.com/User.Read offline_access")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode token response")
}

func TestRefreshToken_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{AccessToken: "tok"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc := oauth.NewService(newRedirectClient(server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := svc.RefreshToken(ctx, "test-tenant", "test-client", "test-secret", "token", "https://graph.microsoft.com/User.Read offline_access")
	assert.Error(t, err)
}

// newRedirectClient creates an HTTP client that redirects all requests to the test server.
func newRedirectClient(testServerURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{targetURL: testServerURL},
	}
}

// redirectTransport rewrites the request URL to point to a test server.
type redirectTransport struct {
	targetURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to our test server
	req.URL.Scheme = "http"
	req.URL.Host = t.targetURL[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}
