package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPublicURLPaths(t *testing.T) {
	for _, test := range []struct {
		value, normalized string
	}{
		{"https://e5.example.com/", "https://e5.example.com"},
		{"https://E5.EXAMPLE.COM:443/tools/e5", "https://e5.example.com/tools/e5"},
		{"http://[::1]:80/e5", "http://[::1]/e5"},
		{"http://192.168.1.10:8080/tools/e5/", "http://192.168.1.10:8080/tools/e5"},
		{"https://e5.example.com/a%20b", ""},
		{"https://e5.example.com/a/../b", ""},
		{"https://e5.example.com/a/./b", ""},
		{"https://e5.example.com/a//b", ""},
		{"https://e5.example.com/a%2fb", ""},
		{"https://e5.example.com/%2e%2e", ""},
		{"https://e5.example.com/a%5cb", ""},
		{"https://e5.example.com/a;b", ""},
		{"https://e5.example.com/a%0ab", ""},
		{"https://e5.example.com/e5?", ""},
		{"https://e5.example.com/e5#", ""},
	} {
		t.Run(test.value, func(t *testing.T) {
			u, err := parsePublicURL(test.value)
			if test.normalized == "" {
				if err == nil {
					t.Fatal("accepted an ambiguous public URL")
				}

				return
			}

			mustNoError(t, err)
			if u.String() != test.normalized {
				t.Fatalf("normalized URL: %s", u)
			}
		})
	}
}

func TestManagementWithBasePath(t *testing.T) {
	for _, prefix := range []string{"", "/e5", "/tools/e5"} {
		t.Run("path="+prefix, func(t *testing.T) {
			site := newTestSite(t)
			config := site.server.config
			config.PublicURL += prefix
			server, err := NewServer(config, site.db, site.runner, site.microsoft, site.server.assets)
			mustNoError(t, err)
			host := httptest.NewServer(server.Handler())
			t.Cleanup(host.Close)
			site.server, site.http = server, host
			client := newBrowser(t, site)

			// Deep links and assets must work before login without exposing a second root entry.
			for _, page := range []string{"/login", "/accounts/42", "/runs?account_id=42", "/settings"} {
				status, body := client.call(t, http.MethodGet, page, nil, "")
				if status != http.StatusOK || !bytes.Contains(body, []byte(`<base href="`+prefix+`/" />`)) {
					t.Fatalf("page %s: %d %s", page, status, body)
				}
			}

			if status, _ := client.call(t, http.MethodGet, "/assets/app.js", nil, ""); status != http.StatusOK {
				t.Fatalf("static asset: %d", status)
			}

			if status, _ := client.call(
				t,
				http.MethodGet,
				"/api/accounts",
				nil,
				"",
			); status != http.StatusUnauthorized {
				t.Fatalf("unprotected management endpoint: %d", status)
			}

			if prefix != "" {
				for _, path := range []string{"/login", "/api/session", "/healthz", prefix + "-other/api/session", prefix + "%2fapi/session"} {
					response, err := client.client.Get(host.URL + path)
					mustNoError(t, err)
					mustNoError(t, response.Body.Close())
					if response.StatusCode != http.StatusNotFound {
						t.Fatalf("outside prefix %s: %d", path, response.StatusCode)
					}
				}

				response, err := client.client.Get(host.URL + prefix + "?source=test")
				mustNoError(t, err)
				mustNoError(t, response.Body.Close())
				if response.StatusCode != http.StatusPermanentRedirect ||
					response.Header.Get("Location") != prefix+"/?source=test" {
					t.Fatal("prefix redirect lost its slash or query")
				}
			}

			// A valid origin excludes the prefix; cookies must still stay inside it.
			client.login(t)
			inside, err := url.Parse(client.host + "/api/session")
			mustNoError(t, err)
			if len(client.client.Jar.Cookies(inside)) == 0 {
				t.Fatal("session cookie does not cover the application")
			}

			outside, err := url.Parse(host.URL + "/elsewhere")
			mustNoError(t, err)
			if prefix != "" && len(client.client.Jar.Cookies(outside)) != 0 {
				t.Fatal("session cookie escaped the configured path")
			}

			if status, _ := client.call(
				t,
				http.MethodPost,
				"/api/logout",
				map[string]string{},
				"https://other.example.com",
			); status != http.StatusForbidden {
				t.Fatal("base path bypassed the origin check")
			}

			// Complete authorization and execution through the same public path.
			account := testAccount(t, site.db)
			status, body := client.call(t, http.MethodPost, "/api/accounts/1/authorize", map[string]string{}, "")
			if status != http.StatusOK {
				t.Fatalf("authorize: %d %s", status, body)
			}

			var target struct{ URL string }
			mustNoError(t, json.Unmarshal(body, &target))
			authorizeURL, err := url.Parse(target.URL)
			mustNoError(t, err)
			callbackURL := config.PublicURL + "/api/oauth/callback"
			if authorizeURL.Query().Get("redirect_uri") != callbackURL {
				t.Fatal("authorization URL lost the base path")
			}

			site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/token") {
					if err := r.ParseForm(); err != nil {
						t.Error(err)
						return nil, err
					}

					if r.Form.Get("grant_type") == "authorization_code" &&
						(r.Form.Get("redirect_uri") != callbackURL || r.Form.Get("code_verifier") == "") {
						t.Error("token exchange lost the redirect URI or PKCE verifier")
					}

					return jsonResponse(
						200,
						`{"access_token":"test-access","refresh_token":"test-refresh","token_type":"Bearer"}`,
					), nil
				}

				return jsonResponse(200, `{}`), nil
			})
			response, err := client.client.Get(
				client.host + "/api/oauth/callback?code=test&state=" + url.QueryEscape(
					authorizeURL.Query().Get("state"),
				),
			)
			mustNoError(t, err)
			mustNoError(t, response.Body.Close())
			if response.StatusCode != http.StatusSeeOther ||
				response.Header.Get("Location") != prefix+"/accounts?authorization=success" {
				t.Fatalf("callback redirect: %d %s", response.StatusCode, response.Header.Get("Location"))
			}

			status, body = client.call(t, http.MethodPost, "/api/accounts/1/run", map[string]string{}, "")
			if status != http.StatusAccepted {
				t.Fatalf("trigger: %d %s", status, body)
			}

			var accepted Run
			mustNoError(t, json.Unmarshal(body, &accepted))
			finished := waitRun(t, site, accepted.ID)
			if finished.Status != RunSuccess || len(finished.Results) == 0 || finished.AccountID != account.ID {
				t.Fatalf("execution did not complete: %+v", finished)
			}

			if status, body := client.call(
				t,
				http.MethodGet,
				"/api/runs?account_id=1",
				nil,
				"",
			); status != http.StatusOK ||
				!bytes.Contains(body, []byte(`"status":"success"`)) {
				t.Fatal("prefixed history did not expose the completed execution")
			}

			if status, _ := client.call(
				t,
				http.MethodPost,
				"/api/logout",
				map[string]string{},
				"",
			); status != http.StatusOK {
				t.Fatal("prefixed logout failed")
			}

			if status, _ := client.call(
				t,
				http.MethodGet,
				"/api/accounts",
				nil,
				"",
			); status != http.StatusUnauthorized {
				t.Fatal("logout retained access")
			}
		})
	}
}
