package executor_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/executor"
	"e5-renewal/backend/services/graph"
	"e5-renewal/backend/services/oauth"
)

func initTestDB(t *testing.T) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	err := database.Init(dsn)
	require.NoError(t, err)
	database.MustInitEncryption("test-encryption-key-minimum16chars")
}

func TestNew(t *testing.T) {
	oauthSvc := oauth.NewService(nil)
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)

	assert.NotNil(t, exec)
	assert.Equal(t, oauthSvc, exec.OAuth)
	assert.NotNil(t, exec.Graph)
	assert.Equal(t, rng, exec.Rand)
}

func TestRun_AuthCodeAccount(t *testing.T) {
	initTestDB(t)

	// Create a mock OAuth server that returns tokens
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "mock-access-token",
			RefreshToken: "mock-new-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	// Create a mock Graph API server
	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServer.URL},
	})
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)
	exec.Graph = &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: graphServer.URL}},
		Rand:       rng,
	}

	// Set up config
	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null") // load defaults with env overrides

	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
		RefreshToken: "old-refresh-token",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:     "test-account",
		AuthType: models.AuthTypeAuthCode,
		AuthInfo: string(authInfoJSON),
	}
	err := database.Accounts.Create(context.Background(), &account)
	require.NoError(t, err)

	taskLog, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	require.NoError(t, err)
	assert.NotNil(t, taskLog)
	assert.Equal(t, account.ID, taskLog.AccountID)
	assert.Equal(t, models.TriggerScheduled, taskLog.TriggerType)
	assert.Greater(t, taskLog.TotalEndpoints, 0)

	// Verify endpoint logs have Scope populated
	eps, err := database.EndpointLogs.ListByTaskLogID(context.Background(), taskLog.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, eps)
	for _, ep := range eps {
		assert.NotEmpty(t, ep.Scope, "EndpointLog.Scope should be populated for endpoint %s", ep.EndpointName)
	}
}

func TestRun_AuthCodeRefreshRotationPreservesAuthExpiry(t *testing.T) {
	initTestDB(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "mock-access-token",
			RefreshToken: "rotated-refresh-token",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServer.URL},
	})
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)
	exec.Graph = &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: graphServer.URL}},
		Rand:       rng,
	}

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")

	originalExpiry := time.Date(2031, time.January, 15, 9, 30, 0, 0, time.UTC)
	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
		RefreshToken: "old-refresh-token",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:          "test-rotation-account",
		AuthType:      models.AuthTypeAuthCode,
		AuthInfo:      string(authInfoJSON),
		AuthExpiresAt: &originalExpiry,
	}
	err := database.Accounts.Create(context.Background(), &account)
	require.NoError(t, err)

	taskLog, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	require.NoError(t, err)
	assert.NotNil(t, taskLog)

	storedAccount, err := database.Accounts.GetByID(context.Background(), account.ID)
	require.NoError(t, err)

	var storedAuthInfo models.AuthInfoData
	require.NoError(t, json.Unmarshal([]byte(storedAccount.AuthInfo), &storedAuthInfo))
	assert.Equal(t, "rotated-refresh-token", storedAuthInfo.RefreshToken)
	require.NotNil(t, storedAccount.AuthExpiresAt)
	assert.True(t, storedAccount.AuthExpiresAt.Equal(originalExpiry), "expected auth_expires_at to remain unchanged")
}

func TestRun_ClientCredentialsAccount(t *testing.T) {
	initTestDB(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:   "Bearer",
			AccessToken: "mock-client-token",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For user resolution and endpoint calls
		if r.URL.Path == "/v1.0/users" {
			json.NewEncoder(w).Encode(map[string]any{
				"value": []map[string]string{{"id": "user-abc"}},
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServer.URL},
	})
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)
	exec.Graph = &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: graphServer.URL}},
		Rand:       rng,
	}

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")

	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:     "test-cc-account",
		AuthType: models.AuthTypeClientCredentials,
		AuthInfo: string(authInfoJSON),
	}
	err := database.Accounts.Create(context.Background(), &account)
	require.NoError(t, err)

	taskLog, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	require.NoError(t, err)
	assert.NotNil(t, taskLog)
	assert.Equal(t, models.AuthTypeClientCredentials, account.AuthType)

	// Verify endpoint logs have Scope populated
	eps, err := database.EndpointLogs.ListByTaskLogID(context.Background(), taskLog.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, eps)
	for _, ep := range eps {
		assert.NotEmpty(t, ep.Scope, "EndpointLog.Scope should be populated for endpoint %s", ep.EndpointName)
	}
}

func TestRun_TokenFailure(t *testing.T) {
	initTestDB(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(oauth.ErrorResponse{
			Error:       "invalid_grant",
			Description: "token expired",
		})
	}))
	defer tokenServer.Close()

	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServer.URL},
	})
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")

	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
		RefreshToken: "expired-token",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:     "test-fail-account",
		AuthType: models.AuthTypeAuthCode,
		AuthInfo: string(authInfoJSON),
	}
	err := database.Accounts.Create(context.Background(), &account)
	require.NoError(t, err)

	taskLog, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "acquire token")
	assert.NotNil(t, taskLog)
	assert.Equal(t, 1, taskLog.FailCount)
}

func TestRun_InvalidAuthInfo(t *testing.T) {
	initTestDB(t)

	account := models.Account{
		Name:     "bad-auth",
		AuthType: models.AuthTypeAuthCode,
		AuthInfo: "not-valid-json",
	}

	oauthSvc := oauth.NewService(nil)
	exec := executor.New(oauthSvc, rand.New(rand.NewSource(42)))

	_, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode auth_info")
}

func TestRunManual_Success(t *testing.T) {
	initTestDB(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := oauth.TokenResponse{
			TokenType:    "Bearer",
			AccessToken:  "manual-token",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	graphServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer graphServer.Close()

	oauthSvc := oauth.NewService(&http.Client{
		Transport: &rewriteTransport{target: tokenServer.URL},
	})
	rng := rand.New(rand.NewSource(42))
	exec := executor.New(oauthSvc, rng)
	exec.Graph = &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: graphServer.URL}},
		Rand:       rng,
	}

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")

	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
		RefreshToken: "refresh-token",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:     "manual-account",
		AuthType: models.AuthTypeAuthCode,
		AuthInfo: string(authInfoJSON),
	}
	err := database.Accounts.Create(context.Background(), &account)
	require.NoError(t, err)

	result, err := exec.RunManual(context.Background(), account)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.TaskLog)
	// RunManual calls all endpoints
	assert.Greater(t, result.TaskLog.TotalEndpoints, 0)

	// Verify endpoint logs have Scope populated
	assert.NotEmpty(t, result.Endpoints)
	for _, ep := range result.Endpoints {
		assert.NotEmpty(t, ep.Scope, "EndpointLog.Scope should be populated for endpoint %s", ep.EndpointName)
	}
}

func TestRun_UnknownAuthType(t *testing.T) {
	initTestDB(t)

	authInfo := models.AuthInfoData{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TenantID:     "test-tenant",
	}
	authInfoJSON, _ := json.Marshal(authInfo)

	account := models.Account{
		Name:     "unknown-auth",
		AuthType: "some_unknown_type",
		AuthInfo: string(authInfoJSON),
	}

	oauthSvc := oauth.NewService(nil)
	exec := executor.New(oauthSvc, rand.New(rand.NewSource(42)))

	t.Setenv("E5_JWT_SECRET", "test-jwt-secret-1234567890")
	t.Setenv("E5_ENCRYPTION_KEY", "test-encryption-key-minimum16chars")
	config.MustInit("/dev/null")

	taskLog, err := exec.Run(context.Background(), account, models.TriggerScheduled)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown auth_type")
	assert.NotNil(t, taskLog)
}

// rewriteTransport redirects all requests to a test server.
type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.target[len("http://"):]
	return http.DefaultTransport.RoundTrip(req)
}
