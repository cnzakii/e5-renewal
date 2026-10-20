package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type Endpoint struct {
	ID       string   `json:"id"`
	Path     string   `json:"-"`
	Scope    string   `json:"scope"`
	AuthType AuthType `json:"auth_type"`
}

// Endpoints is the shared read-only catalog, including each request's required permission.
var Endpoints = []Endpoint{
	{"profile", "/v1.0/me?$select=id,displayName", "User.Read", AuthDelegated},
	{"drive", "/v1.0/me/drive?$select=id", "Files.Read", AuthDelegated},
	{"mail", "/v1.0/me/messages?$top=1&$select=id", "Mail.Read", AuthDelegated},
	{"calendar", "/v1.0/me/calendars?$top=1&$select=id", "Calendars.Read", AuthDelegated},
	{"contacts", "/v1.0/me/contacts?$top=1&$select=id", "Contacts.Read", AuthDelegated},
	{"drive_items", "/v1.0/me/drive/root/children?$top=1&$select=id", "Files.Read", AuthDelegated},
	{"mail_folders", "/v1.0/me/mailFolders?$top=1&$select=id", "Mail.Read", AuthDelegated},
	{"events", "/v1.0/me/events?$top=1&$select=id", "Calendars.Read", AuthDelegated},
	{"contact_folders", "/v1.0/me/contactFolders?$top=1&$select=id", "Contacts.Read", AuthDelegated},
	{"drive_root", "/v1.0/me/drive/root?$select=id", "Files.Read", AuthDelegated},
	{"drives", "/v1.0/me/drives?$top=1&$select=id", "Files.Read", AuthDelegated},
	{"calendar_groups", "/v1.0/me/calendarGroups?$top=1&$select=id", "Calendars.Read", AuthDelegated},
	{"inbox", "/v1.0/me/mailFolders/inbox/messages?$top=1&$select=id", "Mail.Read", AuthDelegated},
	{"sent_mail", "/v1.0/me/mailFolders/sentitems/messages?$top=1&$select=id", "Mail.Read", AuthDelegated},
	{"users", "/v1.0/users?$top=1&$select=id", "User.Read.All", AuthApplication},
	{"organization", "/v1.0/organization?$select=id", "Organization.Read.All", AuthApplication},
	{"sites", "/v1.0/sites/root?$select=id", "Sites.Read.All", AuthApplication},
	{"licenses", "/v1.0/subscribedSkus?$select=skuId", "Organization.Read.All", AuthApplication},
	{"site_drives", "/v1.0/sites/root/drives?$top=1&$select=id", "Sites.Read.All", AuthApplication},
	{"site_lists", "/v1.0/sites/root/lists?$top=1&$select=id", "Sites.Read.All", AuthApplication},
	{
		"deleted_users",
		"/v1.0/directory/deletedItems/microsoft.graph.user?$top=1&$select=id",
		"User.Read.All",
		AuthApplication,
	},
	{"site_columns", "/v1.0/sites/root/columns?$top=1&$select=id", "Sites.Read.All", AuthApplication},
	{"site_content_types", "/v1.0/sites/root/contentTypes?$top=1&$select=id", "Sites.Read.All", AuthApplication},
	{"site_files", "/v1.0/sites/root/drive/root/children?$top=1&$select=id", "Sites.Read.All", AuthApplication},
}

func endpointsFor(authType AuthType) []Endpoint {
	var result []Endpoint
	for _, ep := range Endpoints {
		if ep.AuthType == authType {
			result = append(result, ep)
		}
	}

	return result
}

func scopes(authType AuthType) []string {
	result := []string{"offline_access"}
	for _, ep := range endpointsFor(authType) {
		scope := "https://graph.microsoft.com/" + ep.Scope
		if !slices.Contains(result, scope) {
			result = append(result, scope)
		}
	}

	return result
}

type Microsoft struct {
	Client *http.Client
}

// NewMicrosoft bounds request duration, connection use, and idle connection lifetime.
func NewMicrosoft() *Microsoft {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          8,
		MaxIdleConnsPerHost:   4,
		MaxConnsPerHost:       4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}

	return &Microsoft{Client: &http.Client{
		Transport:     transport,
		Timeout:       30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func (m *Microsoft) oauthConfig(a Account, redirect string) *oauth2.Config {
	base := "https://login.microsoftonline.com/" + a.TenantID + "/oauth2/v2.0/"

	return &oauth2.Config{
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		RedirectURL:  redirect,
		Scopes:       scopes(a.AuthType),
		Endpoint: oauth2.Endpoint{
			AuthURL:   base + "authorize",
			TokenURL:  base + "token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
}

func (m *Microsoft) AuthorizeURL(a Account, redirect, state, verifier string) string {
	return m.oauthConfig(a, redirect).
		AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "select_account"))
}

func (m *Microsoft) Exchange(ctx context.Context, a Account, redirect, code, verifier string) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, m.Client)
	token, err := m.oauthConfig(a, redirect).Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, tokenError(err)
	}

	if token.AccessToken == "" || token.RefreshToken == "" {
		return nil, newError(
			"authorization_tokens_missing",
			"The authorization response is missing tokens. Check the offline_access permission.",
		)
	}

	return token, nil
}

type authFailure struct{ code string }

func (e *authFailure) Error() string {
	return "Microsoft authorization failed. Check the credentials and permissions (" + e.code + ")."
}

func (e *authFailure) ErrorCode() string { return "authorization_" + e.code }

// Token errors expose provider error codes, never response bodies or credentials.
func tokenError(err error) error {
	var response *oauth2.RetrieveError
	if errors.As(err, &response) {
		code := response.ErrorCode
		if code == "" {
			code = "token_request_failed"
		}

		switch code {
		case "invalid_grant",
			"invalid_client",
			"unauthorized_client",
			"invalid_scope",
			"consent_required",
			"interaction_required":
			return &authFailure{code}
		}

		return newError("token_request_failed", fmt.Sprintf("Microsoft token request failed: %s", code))
	}

	return newError("authorization_unavailable", "Could not reach the Microsoft authorization service.")
}

// Token obtains an application token or refreshes a delegated grant.
func (m *Microsoft) Token(ctx context.Context, a Account) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, m.Client)

	var (
		token *oauth2.Token
		err   error
	)
	if a.AuthType == AuthApplication {
		cfg := clientcredentials.Config{
			ClientID: a.ClientID, ClientSecret: a.ClientSecret,
			TokenURL: m.oauthConfig(a, "").Endpoint.TokenURL, Scopes: []string{"https://graph.microsoft.com/.default"},
			AuthStyle: oauth2.AuthStyleInParams,
		}
		token, err = cfg.Token(ctx)
	} else {
		if a.RefreshToken == "" {
			return nil, &authFailure{"missing_refresh_token"}
		}

		token, err = m.oauthConfig(a, "").TokenSource(ctx, &oauth2.Token{RefreshToken: a.RefreshToken}).Token()
	}

	if err != nil {
		return nil, tokenError(err)
	}

	if token.AccessToken == "" {
		return nil, newError("access_token_missing", "The Microsoft response is missing an access token.")
	}

	return token, nil
}

// Call records one Graph request, retrying only bounded throttling and availability failures.
func (m *Microsoft) Call(ctx context.Context, token string, ep Endpoint) (result GraphCall) {
	started := time.Now()
	result = GraphCall{Endpoint: ep.ID, StartedAt: started.UTC()}
	// Keep the monotonic clock for elapsed time and UTC for storage.
	defer func() { result.DurationMS = time.Since(started).Milliseconds() }()
	for attempt := range 3 {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com"+ep.Path, nil)
		if err != nil {
			result.Error = "Could not create the Graph request."
			result.ErrorCode = "graph_request_invalid"
			return result
		}

		req.Header.Set("Authorization", "Bearer "+token)
		result.HTTPStatus = 0
		result.Attempts++
		response, err := m.Client.Do(req)
		if err != nil {
			result.Error = "The Graph request failed."
			result.ErrorCode = "graph_unavailable"
			if ctx.Err() != nil {
				result.Error, result.ErrorCode = "The task was cancelled or timed out.", "task_cancelled"
			}

			return result
		}

		result.HTTPStatus = response.StatusCode
		_, readError := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		if err := response.Body.Close(); err != nil {
			slog.Debug("Graph response close failed", "error_type", fmt.Sprintf("%T", err))
		}

		if readError != nil {
			result.Error, result.ErrorCode = "The Graph response could not be read.", "graph_unavailable"
			if ctx.Err() != nil {
				result.Error, result.ErrorCode = "The task was cancelled or timed out.", "task_cancelled"
			}

			return result
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			result.Error = ""
			result.ErrorCode = ""
			return result
		}

		result.Error = fmt.Sprintf("Graph returned HTTP %d.", response.StatusCode)
		result.ErrorCode = "graph_http_error"
		if (response.StatusCode != 429 && response.StatusCode != 503) || attempt == 2 {
			return result
		}

		delay := time.Duration(attempt+1) * time.Second
		if value := response.Header.Get("Retry-After"); value != "" {
			if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
				if seconds > 30 {
					return result
				}

				delay = time.Duration(seconds) * time.Second
			} else if date, err := http.ParseTime(value); err == nil {
				delay = time.Until(date)
			}
		}

		if delay > 30*time.Second {
			return result
		}

		if delay < 0 {
			delay = 0
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			result.Error = "The task was cancelled or timed out."
			result.ErrorCode = "task_cancelled"
			return result
		case <-timer.C:
		}
	}

	return result
}
