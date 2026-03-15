package graph_test

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e5-renewal/backend/services/graph"
)

func newRng() *rand.Rand {
	return rand.New(rand.NewSource(42))
}

func TestDelegatedScopeURLs(t *testing.T) {
	scopes := graph.DelegatedScopeURLs()
	assert.NotEmpty(t, scopes)

	expected := []string{
		"https://graph.microsoft.com/Files.Read",
		"https://graph.microsoft.com/User.Read.All",
		"https://graph.microsoft.com/User.Read",
		"https://graph.microsoft.com/Mail.Read",
		"https://graph.microsoft.com/MailboxSettings.Read",
		"https://graph.microsoft.com/Calendars.Read",
		"https://graph.microsoft.com/Contacts.Read",
		"https://graph.microsoft.com/Notes.Read",
		"https://graph.microsoft.com/People.Read",
		"https://graph.microsoft.com/Presence.Read",
		"https://graph.microsoft.com/Tasks.Read",
		"https://graph.microsoft.com/Sites.Read.All",
		"https://graph.microsoft.com/Group.Read.All",
		"https://graph.microsoft.com/Organization.Read.All",
	}
	assert.Equal(t, expected, scopes)
}

func TestDelegatedEndpoints(t *testing.T) {
	eps := graph.DelegatedEndpoints()
	assert.NotEmpty(t, eps)

	// Verify structure of first endpoint
	found := false
	for _, ep := range eps {
		assert.NotEmpty(t, ep.Name)
		assert.NotEmpty(t, ep.Path)
		assert.NotEmpty(t, ep.Scope, "endpoint %s must have a Scope", ep.Name)
		if ep.Name == "me/drive/root" {
			found = true
			assert.Equal(t, "/v1.0/me/drive/root", ep.Path)
		}
	}
	assert.True(t, found, "expected to find me/drive/root endpoint")
}

func TestApplicationEndpoints(t *testing.T) {
	eps := graph.ApplicationEndpoints()
	assert.NotEmpty(t, eps)

	// Should contain {userId} placeholders
	hasPlaceholder := false
	for _, ep := range eps {
		assert.NotEmpty(t, ep.Scope, "endpoint %s must have a Scope", ep.Name)
		if ep.Name == "users/{userId}/calendars" {
			hasPlaceholder = true
			assert.Contains(t, ep.Path, "{userId}")
		}
	}
	assert.True(t, hasPlaceholder, "expected to find {userId} placeholder endpoint")
}

func TestEndpointsForAuthType_AuthCode(t *testing.T) {
	eps := graph.EndpointsForAuthType("auth_code")
	delegated := graph.DelegatedEndpoints()
	assert.Equal(t, len(delegated), len(eps))
}

func TestEndpointsForAuthType_ClientCredentials(t *testing.T) {
	eps := graph.EndpointsForAuthType("client_credentials")
	app := graph.ApplicationEndpoints()
	assert.Equal(t, len(app), len(eps))
}

func TestEndpointsForAuthType_Unknown(t *testing.T) {
	eps := graph.EndpointsForAuthType("unknown")
	app := graph.ApplicationEndpoints()
	assert.Equal(t, len(app), len(eps))
}

func TestPickRandomEndpoints_EmptyInput(t *testing.T) {
	r := newRng()
	result := graph.PickRandomEndpoints(nil, 1, 5, r)
	assert.Nil(t, result)
}

func TestPickRandomEndpoints_BasicSelection(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
		{Name: "c", Path: "/c", Scope: "C.Read"},
		{Name: "d", Path: "/d", Scope: "D.Read"},
		{Name: "e", Path: "/e", Scope: "E.Read"},
	}

	r := newRng()
	result := graph.PickRandomEndpoints(endpoints, 2, 4, r)
	assert.GreaterOrEqual(t, len(result), 2)
	assert.LessOrEqual(t, len(result), 4)
}

func TestPickRandomEndpoints_MinEqualsMax(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
		{Name: "c", Path: "/c", Scope: "C.Read"},
	}

	r := newRng()
	result := graph.PickRandomEndpoints(endpoints, 3, 3, r)
	assert.Len(t, result, 3)
}

func TestPickRandomEndpoints_MinGreaterThanMax(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
	}

	r := newRng()
	// maxN < minN => maxN is set to minN
	result := graph.PickRandomEndpoints(endpoints, 2, 1, r)
	assert.Len(t, result, 2)
}

func TestPickRandomEndpoints_MinLessThanOne(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
	}

	r := newRng()
	result := graph.PickRandomEndpoints(endpoints, 0, 1, r)
	assert.Len(t, result, 1)
}

func TestPickRandomEndpoints_MaxExceedsLength(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
	}

	r := newRng()
	result := graph.PickRandomEndpoints(endpoints, 1, 100, r)
	assert.LessOrEqual(t, len(result), len(endpoints))
}

func TestPickRandomEndpoints_NilRand(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
		{Name: "c", Path: "/c", Scope: "C.Read"},
	}

	// nil rand should use fallbackRand
	result := graph.PickRandomEndpoints(endpoints, 1, 3, nil)
	assert.NotEmpty(t, result)
	assert.LessOrEqual(t, len(result), 3)
}

func TestPickRandomEndpoints_DoesNotMutateOriginal(t *testing.T) {
	endpoints := []graph.Endpoint{
		{Name: "a", Path: "/a", Scope: "A.Read"},
		{Name: "b", Path: "/b", Scope: "B.Read"},
		{Name: "c", Path: "/c", Scope: "C.Read"},
	}

	original := make([]graph.Endpoint, len(endpoints))
	copy(original, endpoints)

	r := newRng()
	graph.PickRandomEndpoints(endpoints, 1, 2, r)

	assert.Equal(t, original, endpoints)
}

func TestCallEndpoints_AllSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":[]}`))
	}))
	defer server.Close()

	caller := &graph.Caller{
		HTTPClient: &http.Client{
			Transport: &rewriteTransport{target: server.URL},
		},
		Rand: newRng(),
	}

	endpoints := []graph.Endpoint{
		{Name: "test1", Path: "/v1.0/test1", Scope: "Test1.Read"},
		{Name: "test2", Path: "/v1.0/test2", Scope: "Test2.Read"},
	}

	result := caller.CallEndpoints(context.Background(), "test-token", endpoints, 0, 0)
	assert.Equal(t, 2, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.Results, 2)
	for i, r := range result.Results {
		assert.Equal(t, http.StatusOK, r.Status)
		assert.Empty(t, r.Error)
		assert.Equal(t, endpoints[i].Scope, r.Scope, "Scope should be propagated for endpoint %s", r.Endpoint)
	}
}

func TestCallEndpoints_WithErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.0/fail" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error":"access_denied"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	caller := &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: server.URL}},
		Rand:       newRng(),
	}

	endpoints := []graph.Endpoint{
		{Name: "ok", Path: "/v1.0/ok", Scope: "Ok.Read"},
		{Name: "fail", Path: "/v1.0/fail", Scope: "Fail.Read"},
	}

	result := caller.CallEndpoints(context.Background(), "test-token", endpoints, 0, 0)
	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
}

func TestCallEndpoints_WithUserIDPlaceholder_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1.0/users" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"value": []map[string]string{{"id": "user-123"}},
			})
			return
		}
		// Verify {userId} was replaced
		assert.Contains(t, r.URL.Path, "user-123")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	caller := &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: server.URL}},
		Rand:       newRng(),
	}

	endpoints := []graph.Endpoint{
		{Name: "user-calendar", Path: "/v1.0/users/{userId}/calendars", Scope: "Calendars.Read"},
	}

	result := caller.CallEndpoints(context.Background(), "test-token", endpoints, 0, 0)
	assert.Equal(t, 1, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, "Calendars.Read", result.Results[0].Scope)
}

func TestCallEndpoints_WithUserIDPlaceholder_ResolveFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error for user resolution
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	caller := &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: server.URL}},
		Rand:       newRng(),
	}

	endpoints := []graph.Endpoint{
		{Name: "user-calendar", Path: "/v1.0/users/{userId}/calendars", Scope: "Calendars.Read"},
	}

	result := caller.CallEndpoints(context.Background(), "test-token", endpoints, 0, 0)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 1, result.Failed)
	assert.Contains(t, result.Results[0].Error, "cannot resolve userId")
	assert.Equal(t, "Calendars.Read", result.Results[0].Scope)
}

func TestCallEndpoints_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	caller := &graph.Caller{
		HTTPClient: &http.Client{Transport: &rewriteTransport{target: server.URL}},
		Rand:       newRng(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	endpoints := []graph.Endpoint{
		{Name: "test", Path: "/v1.0/test", Scope: "Test.Read"},
	}

	// With delay > 0 and cancelled context, should return early
	result := caller.CallEndpoints(ctx, "test-token", endpoints, 0, 0)
	// First endpoint has no delay, but the request itself will fail due to cancelled context
	require.Len(t, result.Results, 1)
}

func TestCallEndpoints_Empty(t *testing.T) {
	caller := &graph.Caller{Rand: newRng()}
	result := caller.CallEndpoints(context.Background(), "token", nil, 0, 0)
	assert.Equal(t, 0, result.Succeeded)
	assert.Equal(t, 0, result.Failed)
	assert.Empty(t, result.Results)
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
