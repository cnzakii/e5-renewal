package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const tokenURLTemplate = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"

// Service handles Microsoft OAuth2 token operations.
type Service struct {
	httpClient *http.Client
}

// TokenResponse is the JSON response from the Microsoft token endpoint.
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// ErrorResponse is the JSON error from the Microsoft token endpoint.
type ErrorResponse struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func NewService(httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				ForceAttemptHTTP2: false,
			},
		}
	}
	return &Service{httpClient: httpClient}
}

func (s *Service) BuildAuthorizeURL(tenantID, clientID, redirectURI, state string, scopes []string) string {
	if tenantID == "" {
		tenantID = "common"
	}
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("response_type", "code")
	values.Set("redirect_uri", redirectURI)
	values.Set("response_mode", "query")
	values.Set("scope", strings.Join(scopes, " "))
	values.Set("state", state)
	return fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize?%s", tenantID, values.Encode())
}

// RefreshToken exchanges a refresh_token for a new access_token (and new refresh_token).
// Used by auth_code accounts. The scope parameter specifies the requested permissions.
func (s *Service) RefreshToken(ctx context.Context, tenantID, clientID, clientSecret, refreshToken, scope string) (*TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {scope},
	}
	return s.postToken(ctx, tenantID, form)
}

// AcquireClientToken obtains an access_token using client_credentials flow.
// Used by client_credentials accounts.
func (s *Service) AcquireClientToken(ctx context.Context, tenantID, clientID, clientSecret string) (*TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
		"scope":         {"https://graph.microsoft.com/.default"},
	}
	return s.postToken(ctx, tenantID, form)
}

// ExchangeAuthCode exchanges an authorization code for tokens.
// The scope parameter specifies the requested permissions.
func (s *Service) ExchangeAuthCode(ctx context.Context, tenantID, clientID, clientSecret, code, redirectURI, scope string) (*TokenResponse, error) {
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"scope":         {scope},
	}
	return s.postToken(ctx, tenantID, form)
}

// postToken sends a POST request to the Microsoft token endpoint.
func (s *Service) postToken(ctx context.Context, tenantID string, form url.Values) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf(tokenURLTemplate, tenantID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		_ = json.Unmarshal(body, &errResp)
		return nil, fmt.Errorf("token error %d: %s - %s", resp.StatusCode, errResp.Error, errResp.Description)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tokenResp, nil
}
