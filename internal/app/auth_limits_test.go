package app

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func loginResponse(handler http.Handler, peer, body, forwarded string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/login", strings.NewReader(body))
	request.RemoteAddr = peer
	request.Header.Set("Origin", "http://127.0.0.1:8080")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", forwarded)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	return response
}

func TestLoginQuotaIsIsolatedByClient(t *testing.T) {
	site := newTestSite(t)
	handler := site.server.Handler()
	for range 5 {
		response := loginResponse(handler, "198.51.100.10:1234", `{"password":"wrong"}`, "203.0.113.20")
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("invalid password: %d", response.Code)
		}
	}

	valid := `{"password":"` + testPassword + `"}`
	response := loginResponse(handler, "198.51.100.10:4321", valid, "203.0.113.21")
	if response.Code != http.StatusTooManyRequests {
		t.Fatal("changing the source port or spoofing a forwarding header bypassed the quota")
	}

	if response.Header().Get("Retry-After") != "12" ||
		!strings.Contains(response.Body.String(), `"code":"login_rate_limited"`) {
		t.Fatal("login limit response did not explain how to retry")
	}

	response = loginResponse(handler, "203.0.113.20:1234", valid, "")
	if response.Code != http.StatusOK {
		t.Fatal("one client exhausted another client's login quota")
	}
}

func TestMalformedLoginDoesNotConsumeQuota(t *testing.T) {
	site := newTestSite(t)
	handler := site.server.Handler()
	for range 8 {
		if response := loginResponse(handler, "198.51.100.10:1234", "{", ""); response.Code != http.StatusBadRequest {
			t.Fatalf("malformed login: %d", response.Code)
		}
	}

	response := loginResponse(handler, "198.51.100.10:1234", `{"password":"`+testPassword+`"}`, "")
	if response.Code != http.StatusOK {
		t.Fatal("malformed requests prevented a valid login")
	}
}

func TestPasswordChangeHasIndependentQuota(t *testing.T) {
	site := newTestSite(t)
	client := newBrowser(t, site)
	client.login(t)
	for range 4 {
		client.call(t, http.MethodPost, "/api/login", map[string]string{"password": "wrong"}, "")
	}

	status, _ := client.call(t, http.MethodPost, "/api/login", map[string]string{"password": testPassword}, "")
	if status != http.StatusTooManyRequests {
		t.Fatal("login quota was not exhausted")
	}

	status, _ = client.call(t, http.MethodPut, "/api/password", map[string]string{
		"current_password": testPassword,
		"new_password":     "a-new-administrator-password",
	}, "")
	if status != http.StatusOK {
		t.Fatalf("login traffic blocked a password change: %d", status)
	}

	status, _ = client.call(t, http.MethodGet, "/api/accounts", nil, "")
	if status != http.StatusUnauthorized {
		t.Fatal("password change did not revoke the session")
	}
}

func TestPasswordChangeQuotaDoesNotBlockLogin(t *testing.T) {
	site := newTestSite(t)
	client := newBrowser(t, site)
	client.login(t)
	input := map[string]string{
		"current_password": "wrong",
		"new_password":     "replacement-administrator-password",
	}
	for range 5 {
		status, _ := client.call(t, http.MethodPut, "/api/password", input, "")
		if status != http.StatusBadRequest {
			t.Fatalf("incorrect current password: %d", status)
		}
	}

	status, _ := client.call(t, http.MethodPut, "/api/password", input, "")
	if status != http.StatusTooManyRequests {
		t.Fatal("password change quota was not enforced")
	}

	status, _ = client.call(t, http.MethodPost, "/api/login", map[string]string{"password": testPassword}, "")
	if status != http.StatusOK {
		t.Fatal("password change attempts exhausted the login quota")
	}
}

func TestPasswordCheckCapacity(t *testing.T) {
	site := newTestSite(t)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	releaseChecks := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseChecks)

	var reads atomic.Int32
	mustNoError(
		t,
		site.db.Callback().Query().Before("gorm:query").Register("test:hold-password-check", func(tx *gorm.DB) {
			if tx.Statement.Table != "admins" || reads.Add(1) > 2 {
				return
			}

			entered <- struct{}{}
			<-release
		}),
	)

	handler := site.server.Handler()
	valid := `{"password":"` + testPassword + `"}`
	results := make(chan *httptest.ResponseRecorder, 2)
	for _, peer := range []string{"198.51.100.20:1234", "198.51.100.21:1234"} {
		go func() { results <- loginResponse(handler, peer, valid, "") }()
	}

	for range 2 {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("two password checks could not run concurrently")
		}
	}

	response := loginResponse(handler, "198.51.100.10:1234", valid, "")
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "1" {
		t.Error("password verification did not apply backpressure at capacity")
	}

	releaseChecks()
	for range 2 {
		select {
		case result := <-results:
			if result.Code != http.StatusOK {
				t.Errorf("concurrent login returned %d", result.Code)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("password check did not finish after release")
		}
	}

	response = loginResponse(handler, "198.51.100.10:1234", valid, "")
	if response.Code != http.StatusOK {
		t.Fatal("login did not recover after a password verification slot became available")
	}
}

func TestLoginThroughTrustedProxy(t *testing.T) {
	site := newTestSite(t)
	site.server.config.TrustedProxies = []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	handler := site.server.Handler()
	for range 5 {
		response := loginResponse(handler, "127.0.0.1:1234", `{"password":"wrong"}`, "198.51.100.10")
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("proxied login: %d", response.Code)
		}
	}

	valid := `{"password":"` + testPassword + `"}`
	response := loginResponse(handler, "127.0.0.1:4321", valid, "203.0.113.20, 198.51.100.10")
	if response.Code != http.StatusTooManyRequests {
		t.Fatal("spoofing a hop before the actual client bypassed its quota")
	}

	response = loginResponse(handler, "127.0.0.1:4321", valid, "203.0.113.20")
	if response.Code != http.StatusOK {
		t.Fatal("clients behind one proxy shared a login quota")
	}
}

func TestRestartClearsLoginQuota(t *testing.T) {
	site := newTestSite(t)
	handler := site.server.Handler()
	for range 5 {
		response := loginResponse(handler, "198.51.100.10:1234", `{"password":"wrong"}`, "")
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("invalid password: %d", response.Code)
		}
	}

	valid := `{"password":"` + testPassword + `"}`
	if response := loginResponse(
		handler,
		"198.51.100.10:1234",
		valid,
		"",
	); response.Code != http.StatusTooManyRequests {
		t.Fatal("login quota was not exhausted before restart")
	}

	server, err := NewServer(site.server.config, site.db, site.runner, site.microsoft, site.server.assets)
	mustNoError(t, err)
	response := loginResponse(server.Handler(), "198.51.100.10:1234", valid, "")
	if response.Code != http.StatusOK {
		t.Fatal("restart did not clear the quota or preserve the administrator password")
	}
}

func TestLoginQuotaRecoversOverTime(t *testing.T) {
	limits := newLoginLimiter()
	client := netip.MustParseAddr("198.51.100.10")
	now := time.Now()
	for range 5 {
		if !limits.allow(client, now) {
			t.Fatal("initial login burst was denied")
		}
	}

	if limits.allow(client, now.Add(11*time.Second)) {
		t.Fatal("login quota refilled too early")
	}

	if !limits.allow(client, now.Add(12*time.Second)) || limits.allow(client, now.Add(12*time.Second)) {
		t.Fatal("login quota did not restore exactly one attempt after 12 seconds")
	}
}

func TestForwardedClientRequiresTrustedProxy(t *testing.T) {
	server := &Server{config: Config{TrustedProxies: []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/24"),
		netip.MustParsePrefix("::1/128"),
	}}}
	for _, test := range []struct{ peer, forwarded, expected string }{
		{"198.51.100.10:1234", "203.0.113.20", "198.51.100.10"},
		{"127.0.0.1:1234", "198.51.100.10", "198.51.100.10"},
		{"127.0.0.1:1234", "203.0.113.20, 198.51.100.10", "198.51.100.10"},
		{"127.0.0.1:1234", "198.51.100.10, 127.0.0.2", "198.51.100.10"},
		{"127.0.0.1:1234", "invalid", "127.0.0.1"},
		{"127.0.0.1:1234", "", "127.0.0.1"},
		{"127.0.0.1:1234", "198.51.100.10, invalid", "127.0.0.1"},
		{"127.0.0.1:1234", "127.0.0.2", "127.0.0.1"},
		{"[::1]:1234", "2001:db8::10", "2001:db8::10"},
		{"[::ffff:198.51.100.10]:1234", "203.0.113.20", "198.51.100.10"},
	} {
		request := httptest.NewRequest(http.MethodPost, "/api/login", nil)
		request.RemoteAddr = test.peer
		request.Header.Set("X-Forwarded-For", test.forwarded)
		if got := server.clientIP(request); got.String() != test.expected {
			t.Errorf("peer %s forwarded %q: got %s, want %s", test.peer, test.forwarded, got, test.expected)
		}
	}
}

func TestLoginClientCacheIsBoundedAndExpires(t *testing.T) {
	limits := newLoginLimiter()
	now := time.Now()
	client := netip.MustParseAddr("2001:db8::1")
	for i := range maxLoginClients + 1 {
		if !limits.allow(client, now.Add(time.Duration(i)*time.Nanosecond)) {
			t.Fatal("a full cache denied a new client")
		}

		client = client.Next()
	}

	if len(limits.clients) != maxLoginClients {
		t.Fatalf("client cache size: %d", len(limits.clients))
	}

	if !limits.allow(client, now.Add(loginClientIdle+time.Second)) || len(limits.clients) != 1 {
		t.Fatal("idle client entries did not expire")
	}
}

func TestTrustedProxyConfiguration(t *testing.T) {
	t.Setenv("E5_PUBLIC_URL", "http://localhost:8080")
	t.Setenv("E5_DB_DRIVER", "sqlite")
	t.Setenv("E5_LOG_LEVEL", "INFO")
	t.Setenv("E5_LOG_FORMAT", "text")
	for _, test := range []struct {
		value string
		count int
	}{
		{"", 0},
		{"127.0.0.1, ::1/128, 192.0.2.0/24", 3},
		{"not-an-address", -1},
		{"192.0.2.1/99", -1},
	} {
		t.Setenv("E5_TRUSTED_PROXIES", test.value)
		config, err := LoadConfig()
		if test.count == -1 {
			if err == nil {
				t.Fatal("accepted an invalid trusted proxy")
			}

			continue
		}

		mustNoError(t, err)
		if len(config.TrustedProxies) != test.count {
			t.Fatal("trusted proxy configuration was not loaded")
		}
	}
}
