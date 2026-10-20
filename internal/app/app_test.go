package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"gorm.io/gorm"
)

const (
	testPassword = "a-valid-administrator-password"
	testClientID = "7dfd26ad-11d1-43df-aae8-77c78347a619"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testDB(t testing.TB) *gorm.DB {
	t.Helper()
	directory := t.TempDir()
	db, err := OpenDB(Config{Driver: "sqlite", DataDir: directory, DSN: filepath.Join(directory, "test.db")})
	if err != nil {
		t.Fatal(err)
	}

	sqlDB, err := db.DB()
	mustNoError(t, err)
	t.Cleanup(func() { mustNoError(t, sqlDB.Close()) })
	if _, err := Initialize(db, testPassword); err != nil {
		t.Fatal(err)
	}

	return db
}

func testAccount(t testing.TB, db *gorm.DB) Account {
	t.Helper()
	a := Account{
		Name:         "Example account",
		AuthType:     "delegated",
		ClientID:     testClientID,
		ClientSecret: "secret-never-returned",
		TenantID:     "common",
		RefreshToken: "refresh-never-returned",
		MinMinutes:   120,
		MaxMinutes:   240,
		Revision:     1,
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatal(err)
	}

	return a
}

type testSite struct {
	db        *gorm.DB
	microsoft *Microsoft
	runner    *Runner
	server    *Server
	http      *httptest.Server
}

func newTestSite(t testing.TB) *testSite {
	t.Helper()
	db := testDB(t)
	microsoft := NewMicrosoft()
	microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(
				200,
				`{"access_token":"access-secret","refresh_token":"rotated-secret","expires_in":3600,"token_type":"Bearer"}`,
			), nil
		}

		return jsonResponse(200, "{}"), nil
	})
	runner, err := NewRunner(db, microsoft)
	if err != nil {
		t.Fatal(err)
	}

	// The first delegated sample is profile only, keeping worker tests repeatable.
	runner.random = rand.New(rand.NewPCG(54, 1))
	s, err := NewServer(Config{PublicURL: "http://127.0.0.1:8080"}, db, runner, microsoft, fstest.MapFS{
		"index.html":    {Data: []byte(`<html><head><base href="/" /></head><body>application</body></html>`)},
		"assets/app.js": {Data: []byte("console.log('app')")},
	})
	mustNoError(t, err)
	host := httptest.NewServer(s.Handler())
	t.Cleanup(func() { host.Close(); runner.Close(); microsoft.Client.CloseIdleConnections() })

	return &testSite{db, microsoft, runner, s, host}
}

type browser struct {
	client *http.Client
	host   string
	csrf   string
}

func newBrowser(t testing.TB, site *testSite) *browser {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	b := &browser{
		client: &http.Client{
			Jar:           jar,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		host: site.http.URL + strings.TrimSuffix(site.server.baseHref, "/"),
	}
	status, body := b.call(t, "GET", "/api/session", nil, "")
	if status != 200 {
		t.Fatalf("session: %d %s", status, body)
	}

	var session struct {
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(body, &session); err != nil {
		t.Fatal(err)
	}

	b.csrf = session.CSRF

	return b
}

func (b *browser) call(t testing.TB, method, path string, value any, origin string) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequest(method, b.host+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", b.csrf)
	if origin == "" && method != "GET" {
		origin = "http://127.0.0.1:8080"
	}

	if origin != "" {
		request.Header.Set("Origin", origin)
	}

	response, err := b.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { mustNoError(t, response.Body.Close()) }()
	result, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	return response.StatusCode, result
}

func (b *browser) login(t testing.TB) {
	t.Helper()
	status, body := b.call(t, "POST", "/api/login", map[string]string{"password": testPassword}, "")
	if status != 200 {
		t.Fatalf("login: %d %s", status, body)
	}

	_, body = b.call(t, "GET", "/api/session", nil, "")

	var session struct{ CSRF string }
	if err := json.Unmarshal(body, &session); err != nil {
		t.Fatal(err)
	}

	b.csrf = session.CSRF
}

func TestAdministratorLifecycle(t *testing.T) {
	site := newTestSite(t)
	first, second := newBrowser(t, site), newBrowser(t, site)
	if status, _ := first.call(t, "GET", "/api/accounts", nil, ""); status != 401 {
		t.Fatalf("unauthenticated: %d", status)
	}

	first.login(t)
	second.login(t)
	if password, err := Initialize(site.db, "must-not-replace-existing"); err != nil || password != "" {
		t.Fatalf("restart: %q %v", password, err)
	}

	account := testAccount(t, site.db)
	site.http.Close()
	site.runner.Close()
	if err := ResetPassword(site.db, "another-valid-administrator-password"); err != nil {
		t.Fatal(err)
	}

	runner, err := NewRunner(site.db, site.microsoft)
	mustNoError(t, err)
	server, err := NewServer(site.server.config, site.db, runner, site.microsoft, site.server.assets)
	mustNoError(t, err)
	host := httptest.NewServer(server.Handler())
	t.Cleanup(func() { host.Close(); runner.Close() })
	for _, b := range []*browser{first, second} {
		b.host = host.URL
		if status, _ := b.call(t, "GET", "/api/accounts", nil, ""); status != 401 {
			t.Fatalf("old session remains valid: %d", status)
		}
	}

	if status, _ := first.call(
		t,
		"POST",
		"/api/login",
		map[string]string{"password": "another-valid-administrator-password"},
		"",
	); status != 200 {
		t.Fatalf("replacement password: %d", status)
	}

	var preserved Account
	mustNoError(t, site.db.First(&preserved, account.ID).Error)
	if preserved.ClientSecret != account.ClientSecret {
		t.Fatal("offline administrator recovery changed account credentials")
	}

	if err := ResetPassword(site.db, "short"); err == nil {
		t.Fatal("short password accepted")
	}
}

func TestPasswordChange(t *testing.T) {
	site := newTestSite(t)
	first, second := newBrowser(t, site), newBrowser(t, site)
	first.login(t)
	second.login(t)
	input := map[string]string{"current_password": "wrong", "new_password": "a-new-administrator-password"}
	if status, body := first.call(
		t,
		"PUT",
		"/api/password",
		input,
		"",
	); status != 400 ||
		!bytes.Contains(body, []byte("Incorrect current password.")) {
		t.Fatalf("wrong current password: %d %s", status, body)
	}

	if status, _ := first.call(t, "GET", "/api/accounts", nil, ""); status != 200 {
		t.Fatalf("failed change invalidated the session: %d", status)
	}

	input["current_password"] = testPassword
	if status, body := first.call(t, "PUT", "/api/password", input, ""); status != 200 {
		t.Fatalf("change: %d %s", status, body)
	}

	for _, b := range []*browser{first, second} {
		if status, _ := b.call(t, "GET", "/api/accounts", nil, ""); status != 401 {
			t.Fatalf("old session remains valid: %d", status)
		}
	}

	if status, body := first.call(
		t,
		"POST",
		"/api/login",
		map[string]string{"password": input["new_password"]},
		"",
	); status != 200 {
		t.Fatalf("new password: %d %s", status, body)
	}
}

func TestCSRFLimitsAndRouting(t *testing.T) {
	site := newTestSite(t)
	b := newBrowser(t, site)
	b.login(t)
	token := b.csrf
	b.csrf = "wrong"
	if status, _ := b.call(t, "POST", "/api/accounts", map[string]string{}, ""); status != 403 {
		t.Fatalf("CSRF: %d", status)
	}

	b.csrf = token
	if status, _ := b.call(
		t,
		"POST",
		"/api/login",
		map[string]string{"password": testPassword},
		"https://attacker.example",
	); status != 403 {
		t.Fatalf("origin: %d", status)
	}

	for _, path := range []string{"/api/not-real", "/assets/missing.js", "/.env", "/wp-admin"} {
		if status, _ := b.call(t, "GET", path, nil, ""); status != 404 {
			t.Fatalf("%s: %d", path, status)
		}
	}

	for _, path := range []string{"/accounts", "/runs", "/settings", "/assets/app.js"} {
		if status, _ := b.call(t, "GET", path, nil, ""); status != 200 {
			t.Fatalf("%s: %d", path, status)
		}
	}

	for range 4 {
		b.call(t, "POST", "/api/login", map[string]string{"password": "wrong"}, "")
	}

	if status, _ := b.call(t, "POST", "/api/login", map[string]string{"password": testPassword}, ""); status != 429 {
		t.Fatalf("limit: %d", status)
	}
}

func TestErrorProtocolAndExpiredSession(t *testing.T) {
	site := newTestSite(t)
	b := newBrowser(t, site)
	status, body := b.call(t, "POST", "/api/accounts", map[string]any{}, "")

	var failure struct{ Code, Error string }
	if err := json.Unmarshal(body, &failure); err != nil || status != 401 || failure.Code != "authentication_required" {
		t.Fatalf("expired mutation session: %d %s %v", status, body, err)
	}

	b.login(t)
	status, body = b.call(t, "POST", "/api/accounts", map[string]any{}, "")
	if err := json.Unmarshal(
		body,
		&failure,
	); err != nil || status != 400 || failure.Code != "account_name_invalid" ||
		failure.Error != "The account name must contain 1 to 120 characters." {
		t.Fatalf("validation error contract: %d %s %v", status, body, err)
	}

	b.csrf = "stale-token"
	status, body = b.call(t, "POST", "/api/accounts", map[string]any{}, "")
	if err := json.Unmarshal(body, &failure); err != nil || status != 403 || failure.Code != "csrf_invalid" {
		t.Fatalf("invalid CSRF: %d %s %v", status, body, err)
	}
}

func TestAccountValidationAndSecrets(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	b := newBrowser(t, site)
	b.login(t)
	status, body := b.call(t, "GET", "/api/accounts", nil, "")
	if status != 200 || bytes.Contains(body, []byte(a.ClientSecret)) || bytes.Contains(body, []byte(a.RefreshToken)) {
		t.Fatalf("secret exposure: %d %s", status, body)
	}

	if bytes.Contains(body, []byte(`"endpoints"`)) || site.db.Migrator().HasColumn(&Account{}, "endpoints") {
		t.Fatal("accounts still store or expose their own interface configuration")
	}

	in := accountInput{
		Name:       a.Name,
		AuthType:   a.AuthType,
		ClientID:   a.ClientID,
		TenantID:   a.TenantID,
		MinMinutes: a.MinMinutes,
		MaxMinutes: a.MaxMinutes,
		Revision:   a.Revision,
	}
	in.Name = "Renamed"
	if status, body := b.call(t, "PUT", "/api/accounts/1", in, ""); status != 200 {
		t.Fatalf("update: %d %s", status, body)
	}

	var saved Account
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.ClientSecret != a.ClientSecret || saved.RefreshToken != a.RefreshToken {
		t.Fatal("blank edit destroyed credentials")
	}

	if status, _ := b.call(t, "PUT", "/api/accounts/1", in, ""); status != 409 {
		t.Fatalf("stale edit: %d", status)
	}

	in.Revision = saved.Revision
	in.TenantID = "../../evil?x=1"
	if status, _ := b.call(t, "PUT", "/api/accounts/1", in, ""); status != 400 {
		t.Fatalf("tenant injection: %d", status)
	}

	in.TenantID = "common"
	in.AuthType = "unknown"
	if status, _ := b.call(t, "PUT", "/api/accounts/1", in, ""); status != 400 {
		t.Fatalf("unknown authorization type: %d", status)
	}
}

func TestOAuthPKCEStateConsumption(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	b := newBrowser(t, site)
	b.login(t)
	status, body := b.call(t, "POST", "/api/accounts/1/authorize", map[string]string{}, "")
	if status != 200 {
		t.Fatalf("authorize: %d %s", status, body)
	}

	var target struct {
		URL string `json:"url"`
	}
	mustNoError(t, json.Unmarshal(body, &target))
	u, _ := url.Parse(target.URL)
	query := u.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "" ||
		!strings.Contains(query.Get("scope"), "offline_access") {
		t.Fatal("missing PKCE or refresh scope")
	}

	var calls atomic.Int32
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}

		if r.Form.Get("code_verifier") == "" ||
			r.Form.Get("redirect_uri") != "http://127.0.0.1:8080/api/oauth/callback" {
			t.Error("exchange missing bound verifier or redirect")
		}

		return jsonResponse(200, `{"access_token":"access","refresh_token":"new-refresh","token_type":"Bearer"}`), nil
	})
	callback := "/api/oauth/callback?code=code&state=" + url.QueryEscape(query.Get("state"))
	if status, _ := b.call(t, "GET", callback, nil, ""); status != 303 {
		t.Fatalf("callback: %d", status)
	}

	if status, _ := b.call(t, "GET", callback, nil, ""); status != 303 {
		t.Fatalf("replay: %d", status)
	}

	if calls.Load() != 1 {
		t.Fatal("authorization code was exchanged more than once")
	}

	var saved Account
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.RefreshToken != "new-refresh" {
		t.Fatal("new refresh token not persisted")
	}

	if status, body := b.call(
		t,
		"GET",
		"/api/accounts",
		nil,
		"",
	); status != 200 ||
		bytes.Contains(body, []byte("new-refresh")) {
		t.Fatal("callback token leaked to UI")
	}
}

func TestOAuthDuringExecution(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	started, release := make(chan struct{}), make(chan struct{})
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "graph.microsoft.com" {
			t.Error("stale execution reached Graph")
			return jsonResponse(200, "{}"), nil
		}

		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}

		if r.Form.Get("grant_type") == "authorization_code" {
			// Reauthorization is a new revision even if the token value is unchanged.
			return jsonResponse(
				200,
				`{"access_token":"new-access","refresh_token":"refresh-never-returned","token_type":"Bearer"}`,
			), nil
		}

		close(started)
		select {
		case <-release:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}

		return jsonResponse(
			200,
			`{"access_token":"old-access","refresh_token":"old-job-rotation","token_type":"Bearer"}`,
		), nil
	})
	state := "synthetic-concurrent-authorization"
	attempt := OAuthAttempt{
		StateHash:     stateHash(state),
		AccountID:     a.ID,
		Revision:      a.Revision,
		AdminRevision: 1,
		Verifier:      "test-verifier",
		RedirectURI:   "http://127.0.0.1:8080/api/oauth/callback",
		ExpiresAt:     time.Now().UTC().Add(time.Minute),
	}
	if err := site.db.Create(&attempt).Error; err != nil {
		t.Fatal(err)
	}

	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	<-started
	b := newBrowser(t, site)
	status, _ := b.call(t, "GET", "/api/oauth/callback?code=test&state="+state, nil, "")
	if status != 303 {
		t.Fatalf("callback: %d", status)
	}

	var saved Account
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.RefreshToken != a.RefreshToken || saved.Revision != a.Revision+1 {
		t.Fatal("authorization was not saved during execution")
	}

	close(release)
	waitRun(t, site, run.ID)
	saved = Account{}
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.RefreshToken != a.RefreshToken || saved.Revision != a.Revision+1 {
		t.Fatal("old execution overwrote the new authorization")
	}
}

func TestOAuthRejectsStaleState(t *testing.T) {
	for _, reason := range []string{"expired", "account changed", "administrator reset"} {
		t.Run(reason, func(t *testing.T) {
			site := newTestSite(t)
			a := testAccount(t, site.db)
			state := "a-synthetic-state"
			attempt := OAuthAttempt{
				StateHash:     stateHash(state),
				AccountID:     a.ID,
				Revision:      a.Revision,
				AdminRevision: 1,
				ExpiresAt:     time.Now().UTC().Add(time.Minute),
			}
			switch reason {
			case "expired":
				attempt.ExpiresAt = time.Now().UTC().Add(-time.Minute)
			case "account changed":
				attempt.Revision++
			case "administrator reset":
				attempt.AdminRevision++
			}

			if err := site.db.Create(&attempt).Error; err != nil {
				t.Fatal(err)
			}

			site.microsoft.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
				t.Error("stale state reached the token endpoint")
				return jsonResponse(400, "{}"), nil
			})
			b := newBrowser(t, site)
			status, _ := b.call(t, "GET", "/api/oauth/callback?code=test&state="+state, nil, "")
			if status != 303 {
				t.Fatalf("callback: %d", status)
			}
		})
	}
}

func TestApplicationCredentialFlow(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	a.AuthType, a.TenantID, a.RefreshToken = AuthApplication, "example.onmicrosoft.com", ""
	// The first application sample is directory users only.
	site.runner.random = rand.New(rand.NewPCG(251, 1))
	if err := site.db.Save(&a).Error; err != nil {
		t.Fatal(err)
	}

	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}

			if r.Form.Get("grant_type") != "client_credentials" ||
				r.Form.Get("scope") != "https://graph.microsoft.com/.default" ||
				r.Form.Get("client_secret") != a.ClientSecret ||
				!strings.HasPrefix(r.URL.Path, "/example.onmicrosoft.com/") {
				t.Error("incorrect application credential request")
			}

			return jsonResponse(200, `{"access_token":"application-token","token_type":"Bearer"}`), nil
		}

		if r.URL.Host != "graph.microsoft.com" || r.URL.Path != "/v1.0/users" ||
			r.Header.Get("Authorization") != "Bearer application-token" {
			t.Error("incorrect application Graph request")
		}

		return jsonResponse(200, "{}"), nil
	})
	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	if result := waitRun(t, site, run.ID); result.Status != "success" || len(result.Results) != 1 {
		t.Fatalf("application execution: %+v", result)
	}
}

func waitRun(t testing.TB, site *testSite, id uint) Run {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var run Run
		if site.db.Preload("Results").First(&run, id).Error == nil && run.Status != "running" && !site.runner.Busy() {
			run.measure()
			return run
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("task did not finish")

	return Run{}
}

func TestExecutionLifetimeRotationAndExclusion(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	started, release := make(chan struct{}), make(chan struct{})

	var graphCalls atomic.Int32
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}

			return jsonResponse(
				200,
				`{"access_token":"access","refresh_token":"rotated","token_type":"Bearer","expires_in":3600}`,
			), nil
		}

		graphCalls.Add(1)

		var current Account
		mustNoError(t, site.db.First(&current, a.ID).Error)
		if current.RefreshToken != "rotated" {
			t.Error("Graph ran before token rotation was persisted")
		}

		return jsonResponse(200, "{}"), nil
	})
	b := newBrowser(t, site)
	b.login(t)
	status, body := b.call(t, "POST", "/api/accounts/1/run", map[string]string{}, "")
	if status != 202 {
		t.Fatalf("trigger: %d %s", status, body)
	}

	<-started
	if _, err := site.runner.Trigger(a.ID, "manual"); !errors.Is(err, ErrBusy) {
		t.Fatalf("duplicate execution allowed: %v", err)
	}

	if status, _ := b.call(t, "DELETE", "/api/accounts/1", nil, ""); status != 409 {
		t.Fatalf("delete while running: %d", status)
	}

	close(release)

	var accepted Run
	mustNoError(t, json.Unmarshal(body, &accepted))
	run := waitRun(t, site, accepted.ID)
	if run.Status != "success" || graphCalls.Load() != 1 {
		t.Fatalf("request lifetime leaked into job: %+v", run)
	}
}

func TestRunnerStartupAndShutdown(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	old := Run{AccountID: a.ID, Status: "running", StartedAt: time.Now().UTC().Add(-time.Hour)}
	if err := site.db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}

	if err := site.runner.Start(); err != nil {
		t.Fatal(err)
	}

	var recovered Run
	mustNoError(t, site.db.First(&recovered, old.ID).Error)
	if recovered.Status != "interrupted" || recovered.FinishedAt == nil {
		t.Fatalf("startup recovery: %+v", recovered)
	}

	started := make(chan struct{})
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	<-started
	site.runner.Close()

	var stopped Run
	mustNoError(t, site.db.First(&stopped, run.ID).Error)
	if stopped.Status != "failed" || stopped.FinishedAt == nil {
		t.Fatalf("shutdown result not persisted: %+v", stopped)
	}

	if _, err := site.runner.Trigger(a.ID, "manual"); !errors.Is(err, ErrClosed) {
		t.Fatalf("execution accepted after shutdown: %v", err)
	}
}

func TestExecutionCancellationBetweenCompletedCalls(t *testing.T) {
	for _, test := range []struct {
		name        string
		cancelAfter int
		status      RunStatus
		errorCode   string
	}{
		{"before final call", 1, RunFailed, "task_cancelled"},
		{"after final call", 2, RunSuccess, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			site := newTestSite(t)
			account := testAccount(t, site.db)
			// Select profile followed by mail to cancel between successful calls.
			site.runner.random = rand.New(rand.NewPCG(4915, 1))
			completed := 0
			mustNoError(
				t,
				site.db.Callback().Create().After("gorm:create").Register("test:cancel-after-call", func(tx *gorm.DB) {
					if tx.Statement.Table == "graph_calls" {
						completed++
						if completed == test.cancelAfter {
							site.runner.cancel()
						}
					}
				}),
			)
			run, err := site.runner.Trigger(account.ID, TriggerManual)
			mustNoError(t, err)
			finished := waitRun(t, site, run.ID)
			if finished.Status != test.status || finished.ErrorCode != test.errorCode ||
				finished.RequestCount != test.cancelAfter || finished.SuccessCount != test.cancelAfter {
				t.Fatalf("cancellation lost completed calls or reported the wrong outcome: %+v", finished)
			}
		})
	}
}

func TestRotatedTokenWriteFailureStopsGraph(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)

	var graphCalls atomic.Int32
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(200, `{"access_token":"access","refresh_token":"rotated","token_type":"Bearer"}`), nil
		}

		graphCalls.Add(1)

		return jsonResponse(200, "{}"), nil
	})
	mustNoError(t, site.db.Callback().Update().Before("gorm:update").Register("test:rotation_error", func(tx *gorm.DB) {
		if values, ok := tx.Statement.Dest.(map[string]interface{}); ok {
			if _, exists := values["refresh_token"]; exists {
				tx.Error = errors.New("simulated storage failure")
			}
		}
	}))
	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	result := waitRun(t, site, run.ID)
	if result.Status != "failed" || graphCalls.Load() != 0 {
		t.Fatalf("unsafe continuation: %+v, calls %d", result, graphCalls.Load())
	}
}

func TestScheduleRecoveryAndAuthorizationFailure(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	past := time.Now().UTC().Add(-time.Hour)
	mustNoError(t, site.db.Model(&a).Updates(map[string]any{"enabled": true, "next_run_at": past}).Error)
	site.runner.Tick()

	var run Run
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if site.db.Where("account_id = ?", a.ID).First(&run).Error == nil {
			break
		}

		time.Sleep(5 * time.Millisecond)
	}

	waitRun(t, site, run.ID)
	site.runner.Tick()

	var count int64
	mustNoError(t, site.db.Model(&Run{}).Count(&count).Error)
	if count != 1 {
		t.Fatalf("overdue runs were replayed: %d", count)
	}

	var saved Account
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.NextRunAt == nil || !saved.NextRunAt.After(time.Now()) {
		t.Fatal("next run not persisted")
	}

	site.microsoft.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		return jsonResponse(400, `{"error":"invalid_grant","error_description":"sensitive-detail"}`), nil
	})
	second, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	result := waitRun(t, site, second.ID)
	saved = Account{}
	mustNoError(t, site.db.First(&saved, a.ID).Error)
	if saved.Enabled || saved.NextRunAt != nil || result.Status != "failed" ||
		strings.Contains(result.Error, "sensitive-detail") ||
		result.ErrorCode != "authorization_invalid_grant" ||
		saved.LastErrorCode != result.ErrorCode {
		t.Fatalf("invalid grant handling: %+v %+v", result, saved)
	}
}

func TestGraphRetriesAndCancellation(t *testing.T) {
	m := NewMicrosoft()
	calls := 0
	m.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			response := jsonResponse(429, "{}")
			response.Header.Set("Retry-After", "0")
			return response, nil
		}

		return jsonResponse(200, "{}"), nil
	})
	result := m.Call(context.Background(), "token", Endpoints[0])
	if result.Error != "" || result.ErrorCode != "" || result.HTTPStatus != 200 || calls != 2 || result.Attempts != 2 ||
		result.StartedAt.IsZero() ||
		result.DurationMS < 0 ||
		result.Endpoint != "profile" {
		t.Fatalf("retry result: %+v calls=%d", result, calls)
	}

	calls = 0
	m.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			response := jsonResponse(429, "{}")
			response.Header.Set("Retry-After", "0")
			return response, nil
		}

		return nil, errors.New("network unavailable")
	})
	if result := m.Call(
		context.Background(),
		"token",
		Endpoints[0],
	); result.HTTPStatus != 0 || result.Attempts != 2 ||
		result.ErrorCode != "graph_unavailable" {
		t.Fatalf("network failure retained a previous attempt's status: %+v", result)
	}

	calls = 0
	m.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		response := jsonResponse(429, "{}")
		response.Header.Set("Retry-After", "18446744075")
		return response, nil
	})
	if result := m.Call(context.Background(), "token", Endpoints[0]); result.HTTPStatus != 429 || calls != 1 {
		t.Fatalf("oversized retry delay: %+v calls=%d", result, calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) { return nil, r.Context().Err() })
	if result := m.Call(ctx, "token", Endpoints[0]); result.ErrorCode != "task_cancelled" {
		t.Fatal("cancelled request reported success")
	}
}

func TestExternalDatabaseContract(t *testing.T) {
	for _, driver := range []string{"postgres", "mysql"} {
		t.Run(driver, func(t *testing.T) {
			dsn := os.Getenv("E5_TEST_" + strings.ToUpper(driver) + "_DSN")
			if dsn == "" {
				t.Skip("external database DSN not configured")
			}

			db, err := OpenDB(Config{Driver: driver, DSN: dsn, DataDir: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}

			sqlDB, err := db.DB()
			mustNoError(t, err)
			defer func() { mustNoError(t, sqlDB.Close()) }()
			if !db.Migrator().HasIndex(&Account{}, "idx_accounts_due") ||
				!db.Migrator().HasIndex(&Run{}, "idx_runs_account_started") {
				t.Fatal("required execution indexes are missing")
			}

			if err := db.Create(&GraphCall{Endpoint: "profile", StartedAt: time.Now().UTC()}).Error; err == nil {
				t.Fatal("database accepted an orphan Graph call")
			}

			a := testAccount(t, db)
			defer func() { mustNoError(t, db.Delete(&Account{}, a.ID).Error) }()

			var loaded Account
			if err := db.First(&loaded, a.ID).Error; err != nil {
				t.Fatal(err)
			}

			if loaded.AuthType != a.AuthType || loaded.RefreshToken != a.RefreshToken {
				t.Fatal("database serialization mismatch")
			}

			input := accountInput{
				Name:       "Updated",
				AuthType:   a.AuthType,
				ClientID:   a.ClientID,
				TenantID:   a.TenantID,
				MinMinutes: 2,
				MaxMinutes: 3,
				Enabled:    true,
			}
			if err := input.apply(&loaded); err != nil {
				t.Fatal(err)
			}

			if err := db.Save(&loaded).Error; err != nil {
				t.Fatal(err)
			}

			var updated Account
			mustNoError(t, db.First(&updated, a.ID).Error)
			if updated.NextRunAt == nil || !updated.Enabled {
				t.Fatal("schedule persistence mismatch")
			}

			run := Run{
				AccountID:   a.ID,
				AccountName: a.Name,
				Trigger:     "manual",
				Status:      "failed",
				Error:       "Graph requests failed.",
				ErrorCode:   "graph_requests_failed",
				StartedAt:   time.Now().UTC(),
				Results: []GraphCall{
					{
						Endpoint:   "profile",
						StartedAt:  time.Now().UTC(),
						Attempts:   1,
						HTTPStatus: 403,
						Error:      "Graph returned HTTP 403.",
						ErrorCode:  "graph_http_error",
					},
				},
			}
			run.measure()
			if err := db.Create(&run).Error; err != nil {
				t.Fatal(err)
			}

			defer func() {
				mustNoError(t, db.Where("run_id = ?", run.ID).Delete(&GraphCall{}).Error)
				mustNoError(t, db.Delete(&Run{}, run.ID).Error)
			}()

			var stored Run
			if err := db.Preload("Results").
				First(&stored, run.ID).
				Error; err != nil || stored.ErrorCode != run.ErrorCode || len(stored.Results) != 1 ||
				stored.Results[0].ErrorCode != "graph_http_error" {
				t.Fatalf("execution error serialization: %+v %v", stored, err)
			}

			statistics, err := (&Server{db: db}).statistics(
				context.Background(),
				a.ID,
				statisticsStart(time.Now(), 7, time.UTC),
			)
			if err != nil || statistics[a.ID].Requests != 1 || len(statistics[a.ID].Daily) != 1 ||
				statistics[a.ID].Daily[0].Day != time.Now().UTC().Format("2006-01-02") {
				t.Fatalf("database statistics: %+v %v", statistics, err)
			}

			if _, err := Initialize(db, testPassword); err != nil {
				t.Fatal(err)
			}

			var admin Admin
			mustNoError(t, db.First(&admin, 1).Error)
			state := randomToken()
			attempt := OAuthAttempt{
				StateHash:     stateHash(state),
				AccountID:     a.ID,
				Revision:      updated.Revision,
				AdminRevision: admin.Revision,
				Verifier:      "database-fixture",
				RedirectURI:   "https://callback.example.com/return",
				ExpiresAt:     time.Now().UTC().Add(time.Minute),
			}
			if err := db.Create(&attempt).Error; err != nil {
				t.Fatal(err)
			}

			defer db.Delete(&OAuthAttempt{}, "state_hash = ?", attempt.StateHash)
			microsoft := NewMicrosoft()
			microsoft.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(
					200,
					`{"access_token":"database-access","refresh_token":"database-refresh","expires_in":3600}`,
				), nil
			})
			if err := (&Server{db: db, microsoft: microsoft}).completeAuthorization(
				context.Background(),
				state,
				"fixture-code",
				attempt.RedirectURI,
				a.ID,
			); err != nil {
				t.Fatalf("database OAuth completion: %v", err)
			}
		})
	}
}

func TestAccountMetadataPreservesCredentials(t *testing.T) {
	a := Account{}
	secret := "fixture-secret"
	in := accountInput{
		Name:         "New",
		AuthType:     "delegated",
		TenantID:     "common",
		ClientID:     testClientID,
		ClientSecret: &secret,
		MinMinutes:   120,
		MaxMinutes:   240,
	}
	if err := in.apply(&a); err != nil || a.AuthType != AuthDelegated {
		t.Fatalf("new account: %+v %v", a, err)
	}

	a.RefreshToken = "keep-token"
	in.ClientSecret = nil
	in.Name = "Renamed"
	if err := in.apply(&a); err != nil || a.RefreshToken != "keep-token" {
		t.Fatalf("credential preservation: %+v %v", a, err)
	}

	in.AuthType = "application"
	in.TenantID = "example.onmicrosoft.com"
	if err := in.apply(&a); err != nil || a.AuthType != AuthApplication || a.RefreshToken != "" {
		t.Fatalf("authorization type change: %+v %v", a, err)
	}
}

func TestManualCallbackBindingAndSingleUse(t *testing.T) {
	site := newTestSite(t)
	testAccount(t, site.db)
	b := newBrowser(t, site)
	b.login(t)
	redirect := "http://callback.example.com/return"
	status, body := b.call(t, "POST", "/api/accounts/1/authorize", map[string]string{"redirect_uri": redirect}, "")

	var authorization struct{ URL string }
	mustNoError(t, json.Unmarshal(body, &authorization))
	target, _ := url.Parse(authorization.URL)
	state := target.Query().Get("state")
	if status != 200 || state == "" || target.Query().Get("redirect_uri") != redirect {
		t.Fatalf("authorize: %d %s", status, body)
	}

	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}

		if r.Form.Get("redirect_uri") != redirect || r.Form.Get("code_verifier") == "" {
			t.Error("manual exchange lost redirect or PKCE")
		}

		return jsonResponse(
			200,
			`{"access_token":"access-sensitive","refresh_token":"refresh-sensitive","expires_in":3600}`,
		), nil
	})
	callback := redirect + "?code=test-code&state=" + state
	if status, _ := b.call(
		t,
		"POST",
		"/api/accounts/1/exchange",
		map[string]string{"callback_url": "https://other.example/return?code=test-code&state=" + state},
		"",
	); status != 400 {
		t.Fatalf("wrong redirect: %d", status)
	}

	if status, _ := b.call(
		t,
		"POST",
		"/api/accounts/2/exchange",
		map[string]string{"callback_url": callback},
		"",
	); status != 400 {
		t.Fatalf("wrong account: %d", status)
	}

	status, body = b.call(t, "POST", "/api/accounts/1/exchange", map[string]string{"callback_url": callback}, "")
	if status != 200 || bytes.Contains(body, []byte("sensitive")) {
		t.Fatalf("exchange result: %d %s", status, body)
	}

	if status, _ := b.call(
		t,
		"POST",
		"/api/accounts/1/exchange",
		map[string]string{"callback_url": callback},
		"",
	); status != 400 {
		t.Fatalf("reused state: %d", status)
	}

	var saved Account
	mustNoError(t, site.db.First(&saved, 1).Error)
	if saved.RefreshToken != "refresh-sensitive" {
		t.Fatal("manual exchange did not save the refresh token")
	}
}

func TestPartialExecutionStatisticsAndFilters(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	// Select profile followed by mail to exercise a partial execution.
	site.runner.random = rand.New(rand.NewPCG(4915, 1))
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(200, `{"access_token":"access","refresh_token":"rotated","expires_in":3600}`), nil
		}

		if strings.Contains(r.URL.Path, "messages") {
			return jsonResponse(403, `{}`), nil
		}

		return jsonResponse(200, `{}`), nil
	})
	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	saved := waitRun(t, site, run.ID)
	if saved.Status != "partial" || saved.RequestCount != 2 || saved.SuccessCount != 1 {
		t.Fatalf("partial result: %+v", saved)
	}

	b := newBrowser(t, site)
	b.login(t)
	status, body := b.call(t, "GET", "/api/accounts/1", nil, "")

	var view struct{ Statistics AccountStatistics }
	mustNoError(t, json.Unmarshal(body, &view))
	if status != 200 || view.Statistics.Requests != 2 || view.Statistics.Successes != 1 ||
		len(view.Statistics.Daily) != 7 {
		t.Fatalf("statistics: %d %s", status, body)
	}

	for _, path := range []string{"/api/runs?status=partial&trigger=manual&days=7", "/api/runs?account_id=1&status=partial"} {
		status, body = b.call(t, "GET", path, nil, "")

		var page struct {
			Runs  []Run
			Total int
			Page  int
		}
		mustNoError(t, json.Unmarshal(body, &page))
		if status != 200 || page.Total != 1 || page.Page != 1 || len(page.Runs) != 1 {
			t.Fatalf("filtered history: %d %s", status, body)
		}
	}

	status, body = b.call(t, "GET", "/api/runs?status=success", nil, "")
	if status != 200 || !bytes.Contains(body, []byte(`"total":0`)) {
		t.Fatalf("success filter: %d %s", status, body)
	}

	if status, _ := b.call(t, "GET", "/api/runs?days=999", nil, ""); status != 400 {
		t.Fatalf("unbounded time range: %d", status)
	}

	if status, body := b.call(t, "DELETE", "/api/accounts/1", nil, ""); status != 204 {
		t.Fatalf("account deletion: %d %s", status, body)
	}

	var count int64
	mustNoError(t, site.db.Model(&GraphCall{}).Count(&count).Error)
	if count != 0 {
		t.Fatal("account deletion left Graph call records")
	}

	mustNoError(t, site.db.Model(&Run{}).Count(&count).Error)
	if count != 0 {
		t.Fatal("account deletion left execution records")
	}
}

func TestRetentionKeepsRecentHistoryBeyondOneThousand(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	recent := make([]Run, 1005)
	for i := range recent {
		recent[i] = Run{
			AccountID:   a.ID,
			AccountName: a.Name,
			Status:      "success",
			Trigger:     "manual",
			StartedAt:   time.Now().UTC(),
			Results:     []GraphCall{},
		}
		recent[i].measure()
	}

	if err := site.db.CreateInBatches(&recent, 100).Error; err != nil {
		t.Fatal(err)
	}

	old := Run{
		AccountID: a.ID,
		Status:    "success",
		StartedAt: time.Now().UTC().AddDate(0, 0, -31),
		Results:   []GraphCall{{Endpoint: "profile", HTTPStatus: 200, StartedAt: time.Now().UTC().AddDate(0, 0, -31)}},
	}
	old.measure()
	mustNoError(t, site.db.Create(&old).Error)
	site.runner.prune()

	var count int64
	mustNoError(t, site.db.Model(&Run{}).Count(&count).Error)

	var callCount int64
	mustNoError(t, site.db.Model(&GraphCall{}).Count(&callCount).Error)
	if callCount != 0 {
		t.Fatal("retention left orphan Graph calls")
	}

	if count != 1005 {
		t.Fatalf("recent history was truncated: %d", count)
	}

	b := newBrowser(t, site)
	b.login(t)
	status, body := b.call(t, "GET", "/api/runs?page=2", nil, "")

	var page struct {
		Runs     []Run
		Total    int
		Page     int
		PageSize int `json:"page_size"`
	}
	mustNoError(t, json.Unmarshal(body, &page))
	if status != 200 || page.Total != 1005 || page.Page != 2 || page.PageSize != 25 || len(page.Runs) != 25 {
		t.Fatalf("pagination: %d %s", status, body)
	}
}

func TestGraphCallsPersistBeforeRunFinishes(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	// Select profile followed by mail so the second request can be held open.
	site.runner.random = rand.New(rand.NewPCG(4915, 1))
	entered, release := make(chan struct{}), make(chan struct{})

	var once sync.Once
	defer once.Do(func() { close(release) })
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(200, `{"access_token":"test-access","expires_in":3600}`), nil
		}

		if strings.Contains(r.URL.Path, "messages") {
			close(entered)
			<-release
		}

		return jsonResponse(200, `{}`), nil
	})
	run, err := site.runner.Trigger(a.ID, "manual")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("second Graph call did not start")
	}

	var calls []GraphCall
	mustNoError(t, site.db.Where("run_id = ?", run.ID).Find(&calls).Error)
	if len(calls) != 1 || calls[0].Endpoint != "profile" || calls[0].StartedAt.IsZero() || calls[0].Attempts != 1 {
		t.Fatalf("first call was not persisted independently: %+v", calls)
	}

	var running Run
	mustNoError(t, site.db.First(&running, run.ID).Error)
	if running.Status != "running" {
		t.Fatal("run completed before the blocked call")
	}

	once.Do(func() { close(release) })
	finished := waitRun(t, site, run.ID)
	if finished.RequestCount != 2 || finished.SuccessCount != 2 || finished.Results[0].ID == finished.Results[1].ID {
		t.Fatalf("completed calls: %+v", finished)
	}
}

func TestGraphStatisticsUseCallTime(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	now := time.Now().UTC()
	run := Run{AccountID: a.ID, Status: "partial", StartedAt: now.AddDate(0, 0, -8), Results: []GraphCall{
		{Endpoint: "profile", HTTPStatus: 200, StartedAt: now},
		{Endpoint: "mail", HTTPStatus: 403, Error: "Graph returned HTTP 403.", StartedAt: now.AddDate(0, 0, -8)},
	}}
	if err := site.db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	statistics, err := site.server.statistics(context.Background(), a.ID, statisticsStart(time.Now(), 7, time.UTC))
	value := fillStatistics(statistics[a.ID], statisticsStart(time.Now(), 7, time.UTC))
	if err != nil || value.Requests != 1 || value.Successes != 1 || len(value.Daily) != 7 ||
		value.Daily[6].Requests != 1 {
		t.Fatalf("statistics did not use individual call timestamps: %+v %v", value, err)
	}
}

func TestLocalCalendarStatisticsAndDrilldown(t *testing.T) {
	site := newTestSite(t)
	account := testAccount(t, site.db)
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	start := statisticsStart(time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), 7, location)
	if start.AddDate(0, 0, 6).Sub(start.AddDate(0, 0, 5)) != 23*time.Hour {
		t.Fatal("DST boundary must use a calendar day")
	}

	run := Run{
		AccountID:   account.ID,
		AccountName: account.Name,
		Status:      "success",
		StartedAt:   time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC),
		Results: []GraphCall{
			{Endpoint: "profile", HTTPStatus: 200, StartedAt: time.Date(2026, 3, 8, 4, 59, 59, 0, time.UTC)},
			{Endpoint: "profile", HTTPStatus: 200, StartedAt: time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC)},
			{
				Endpoint:   "profile",
				HTTPStatus: 403,
				Error:      "Graph returned HTTP 403.",
				StartedAt:  time.Date(2026, 3, 9, 4, 0, 0, 0, time.UTC),
			},
		},
	}
	if err := site.db.Create(&run).Error; err != nil {
		t.Fatal(err)
	}

	statistics, err := site.server.statistics(context.Background(), account.ID, start)
	value := fillStatistics(statistics[account.ID], start)
	if err != nil || value.Requests != 3 || value.Successes != 2 || value.Daily[4].Requests != 1 ||
		value.Daily[5].Requests != 1 ||
		value.Daily[6].Requests != 1 {
		t.Fatalf("local calendar grouping: %+v %v", value, err)
	}

	browser := newBrowser(t, site)
	browser.login(t)
	for _, fixture := range []struct {
		path, zone string
		status     int
		total      int
	}{
		{"/api/runs?date=2026-03-08", "America/New_York", 200, 1},
		{"/api/runs?date=2026-03-10", "America/New_York", 200, 0},
		{"/api/runs?date=2026-02-30", "America/New_York", 400, 0},
		{"/api/accounts", "invalid/time-zone", 400, 0},
	} {
		request, _ := http.NewRequest("GET", site.http.URL+fixture.path, nil)
		request.Header.Set("X-Time-Zone", fixture.zone)
		response, err := browser.client.Do(request)
		if err != nil {
			t.Fatal(err)
		}

		var body struct{ Total int }
		mustNoError(t, json.NewDecoder(response.Body).Decode(&body))
		mustNoError(t, response.Body.Close())
		if response.StatusCode != fixture.status || body.Total != fixture.total {
			t.Fatalf("%s: %d %+v", fixture.path, response.StatusCode, body)
		}
	}
}
