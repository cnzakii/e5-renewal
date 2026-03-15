package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const BaseURL = "https://graph.microsoft.com"

// Endpoint defines a single Microsoft Graph API endpoint.
type Endpoint struct {
	Name  string
	Path  string
	Scope string
}

// EndpointResult records the outcome of a single endpoint call.
type EndpointResult struct {
	Endpoint     string
	Scope        string
	Status       int
	Error        string
	ResponseBody string // populated on HTTP 4xx/5xx responses
	ExecutedAt   time.Time
}

// RunResult summarises all endpoint calls in one task execution.
type RunResult struct {
	Succeeded int
	Failed    int
	Results   []EndpointResult
}

// Caller makes real HTTP requests to the Microsoft Graph API.
type Caller struct {
	HTTPClient *http.Client
	Rand       *rand.Rand
}

// DelegatedScopeURLs returns the unique full scope URLs derived from delegated endpoints.
func DelegatedScopeURLs() []string {
	seen := map[string]struct{}{}
	var urls []string
	for _, ep := range DelegatedEndpoints() {
		if ep.Scope == "" {
			continue
		}
		u := BaseURL + "/" + ep.Scope
		if _, ok := seen[u]; !ok {
			seen[u] = struct{}{}
			urls = append(urls, u)
		}
	}
	return urls
}

// DelegatedEndpoints returns endpoints requiring user context (auth_code).
func DelegatedEndpoints() []Endpoint {
	return []Endpoint{
		{Name: "me/drive/root", Path: "/v1.0/me/drive/root", Scope: "Files.Read"},
		{Name: "me/drive", Path: "/v1.0/me/drive", Scope: "Files.Read"},
		{Name: "drive/root", Path: "/v1.0/drive/root", Scope: "Files.Read"},
		{Name: "me/drive/root/children", Path: "/v1.0/me/drive/root/children?$top=1", Scope: "Files.Read"},
		{Name: "users", Path: "/v1.0/users?$top=1", Scope: "User.Read.All"},
		{Name: "me/select", Path: "/v1.0/me?$select=displayName,skills", Scope: "User.Read"},
		{Name: "me/messages", Path: "/v1.0/me/messages?$top=1", Scope: "Mail.Read"},
		{Name: "me/mailFolders", Path: "/v1.0/me/mailFolders", Scope: "Mail.Read"},
		{Name: "me/mailFolders/inbox/rules", Path: "/v1.0/me/mailFolders/inbox/messageRules", Scope: "MailboxSettings.Read"},
		{Name: "me/mailFolders/inbox/delta", Path: "/v1.0/me/mailFolders/Inbox/messages/delta?$top=1", Scope: "Mail.Read"},
		{Name: "me/outlook/categories", Path: "/v1.0/me/outlook/masterCategories", Scope: "MailboxSettings.Read"},
		{Name: "me/calendars", Path: "/v1.0/me/calendars?$top=1", Scope: "Calendars.Read"},
		{Name: "me/contacts", Path: "/v1.0/me/contacts?$top=1", Scope: "Contacts.Read"},
		{Name: "me/mailboxSettings", Path: "/v1.0/me/mailboxSettings", Scope: "MailboxSettings.Read"},
		{Name: "me/onenote/notebooks", Path: "/v1.0/me/onenote/notebooks?$top=1", Scope: "Notes.Read"},
		{Name: "me/people", Path: "/v1.0/me/people?$top=1", Scope: "People.Read"},
		{Name: "me/presence", Path: "/v1.0/me/presence", Scope: "Presence.Read"},
		{Name: "me/todo/lists", Path: "/v1.0/me/todo/lists", Scope: "Tasks.Read"},
		{Name: "sites/root", Path: "/v1.0/sites/root", Scope: "Sites.Read.All"},
		{Name: "sites/root/lists", Path: "/v1.0/sites/root/lists", Scope: "Sites.Read.All"},
		{Name: "sites/root/drives", Path: "/v1.0/sites/root/drives", Scope: "Sites.Read.All"},
		{Name: "groups", Path: "/v1.0/groups?$top=1", Scope: "Group.Read.All"},
		{Name: "organization", Path: "/v1.0/organization", Scope: "Organization.Read.All"},
	}
}

// ApplicationEndpoints returns endpoints for client_credentials flow.
// Endpoints with {userId} are resolved at call time via /v1.0/users?$top=1.
func ApplicationEndpoints() []Endpoint {
	return []Endpoint{
		{Name: "users", Path: "/v1.0/users?$top=1", Scope: "User.Read.All"},
		{Name: "organization", Path: "/v1.0/organization", Scope: "Organization.Read.All"},
		{Name: "sites/root", Path: "/v1.0/sites/root", Scope: "Sites.Read.All"},
		{Name: "sites/root/lists", Path: "/v1.0/sites/root/lists", Scope: "Sites.Read.All"},
		{Name: "sites/root/drives", Path: "/v1.0/sites/root/drives", Scope: "Sites.Read.All"},
		{Name: "users/{userId}/calendars", Path: "/v1.0/users/{userId}/calendars?$top=1", Scope: "Calendars.Read"},
		{Name: "users/{userId}/contacts", Path: "/v1.0/users/{userId}/contacts?$top=1", Scope: "Contacts.Read"},
		{Name: "users/{userId}/drive/root", Path: "/v1.0/users/{userId}/drive/root", Scope: "Files.Read.All"},
		{Name: "users/{userId}/drive/root/children", Path: "/v1.0/users/{userId}/drive/root/children?$top=1", Scope: "Files.Read.All"},
		{Name: "users/{userId}/messages", Path: "/v1.0/users/{userId}/messages?$top=1", Scope: "Mail.Read"},
		{Name: "users/{userId}/mailboxSettings", Path: "/v1.0/users/{userId}/mailboxSettings", Scope: "MailboxSettings.Read"},
		{Name: "users/{userId}/onenote/notebooks", Path: "/v1.0/users/{userId}/onenote/notebooks?$top=1", Scope: "Notes.Read"},
		{Name: "groups", Path: "/v1.0/groups?$top=1", Scope: "Group.Read.All"},
	}
}

// EndpointsForAuthType returns the endpoint pool for a given auth_type.
func EndpointsForAuthType(authType string) []Endpoint {
	if authType == "auth_code" {
		return DelegatedEndpoints()
	}
	return ApplicationEndpoints()
}

// PickRandomEndpoints selects a random subset of endpoints.
func PickRandomEndpoints(all []Endpoint, minN, maxN int, r *rand.Rand) []Endpoint {
	if len(all) == 0 {
		return nil
	}
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	if minN < 1 {
		minN = 1
	}
	if maxN < minN {
		maxN = minN
	}
	if maxN > len(all) {
		maxN = len(all)
	}
	if minN > len(all) {
		minN = len(all)
	}

	n := minN
	if maxN > minN {
		n = minN + r.Intn(maxN-minN+1)
	}

	picked := append([]Endpoint(nil), all...)
	r.Shuffle(len(picked), func(i, j int) {
		picked[i], picked[j] = picked[j], picked[i]
	})
	return picked[:n]
}

// CallEndpoints calls each endpoint with real HTTP GET requests.
// For application mode, {userId} placeholders are resolved by calling /v1.0/users?$top=1 first.
// Delays randomly between minDelaySec and maxDelaySec seconds between calls.
func (c *Caller) CallEndpoints(ctx context.Context, token string, endpoints []Endpoint, minDelaySec, maxDelaySec int) RunResult {
	result := RunResult{Results: make([]EndpointResult, 0, len(endpoints))}
	r := c.Rand
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// Resolve {userId} if any endpoint needs it
	var userID string
	for _, ep := range endpoints {
		if strings.Contains(ep.Path, "{userId}") {
			userID = c.resolveUserID(ctx, token)
			break
		}
	}

	for i, ep := range endpoints {
		// Delay between calls (not before the first one)
		if i > 0 && minDelaySec > 0 {
			delay := minDelaySec
			if maxDelaySec > minDelaySec {
				delay += r.Intn(maxDelaySec - minDelaySec + 1)
			}
			select {
			case <-ctx.Done():
				return result
			case <-time.After(time.Duration(delay) * time.Second):
			}
		}

		// Replace {userId} placeholder
		if strings.Contains(ep.Path, "{userId}") {
			if userID == "" {
				result.Results = append(result.Results, EndpointResult{
					Endpoint:   ep.Name,
					Scope:      ep.Scope,
					Status:     0,
					Error:      "cannot resolve userId: User.Read.All failed",
					ExecutedAt: time.Now().UTC(),
				})
				result.Failed++
				continue
			}
			ep.Path = strings.ReplaceAll(ep.Path, "{userId}", userID)
		}

		er := c.callOne(ctx, token, ep)
		result.Results = append(result.Results, er)
		if er.Error == "" {
			result.Succeeded++
		} else {
			result.Failed++
		}
	}
	return result
}

// resolveUserID calls /v1.0/users?$top=1 and extracts the first user's ID.
func (c *Caller) resolveUserID(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+"/v1.0/users?$top=1", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return ""
	}

	var body struct {
		Value []struct {
			ID string `json:"id"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || len(body.Value) == 0 {
		return ""
	}
	return body.Value[0].ID
}

func (c *Caller) callOne(ctx context.Context, token string, ep Endpoint) EndpointResult {
	er := EndpointResult{Endpoint: ep.Name, Scope: ep.Scope, ExecutedAt: time.Now().UTC()}

	url := BaseURL + ep.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		er.Status = 0
		er.Error = fmt.Sprintf("build request: %v", err)
		return er
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		er.Status = 0
		er.Error = fmt.Sprintf("request failed: %v", err)
		return er
	}
	defer resp.Body.Close()

	er.Status = resp.StatusCode
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		_, _ = io.Copy(io.Discard, resp.Body) // drain remaining for connection reuse
		er.ResponseBody = string(bodyBytes)
		er.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}
	return er
}
