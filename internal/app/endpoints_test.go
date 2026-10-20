package app

import (
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSharedEndpointPermissions(t *testing.T) {
	for _, tc := range []struct {
		authType AuthType
		ids      []string
		scopes   []string
	}{
		{
			AuthDelegated,
			[]string{
				"profile", "drive", "mail", "calendar", "contacts", "drive_items", "mail_folders", "events", "contact_folders",
				"drive_root", "drives", "calendar_groups", "inbox", "sent_mail",
			},
			[]string{"offline_access", "User.Read", "Files.Read", "Mail.Read", "Calendars.Read", "Contacts.Read"},
		},
		{
			AuthApplication,
			[]string{
				"users", "organization", "sites", "licenses", "site_drives", "site_lists",
				"deleted_users", "site_columns", "site_content_types", "site_files",
			},
			[]string{"offline_access", "User.Read.All", "Organization.Read.All", "Sites.Read.All"},
		},
	} {
		t.Run(string(tc.authType), func(t *testing.T) {
			var ids []string
			for _, ep := range endpointsFor(tc.authType) {
				ids = append(ids, ep.ID)
				u, err := url.Parse(ep.Path)
				if err != nil || u.IsAbs() || !strings.HasPrefix(u.Path, "/v1.0/") || u.Query().Get("$select") == "" {
					t.Fatalf("unbounded or invalid Graph interface: %+v", ep)
				}
			}

			if !slices.Equal(ids, tc.ids) {
				t.Fatalf("shared pool: %v", ids)
			}

			wantScopes := slices.Clone(tc.scopes)
			for i := 1; i < len(wantScopes); i++ {
				wantScopes[i] = "https://graph.microsoft.com/" + wantScopes[i]
			}

			if got := scopes(tc.authType); !slices.Equal(got, wantScopes) {
				t.Fatalf("permissions were missing, broadened, or duplicated: %v", got)
			}
		})
	}

	account := Account{AuthType: AuthDelegated, TenantID: "common", ClientID: testClientID}
	target, err := url.Parse(NewMicrosoft().AuthorizeURL(account, "https://example.com/callback", "state", "verifier"))
	mustNoError(t, err)
	if !slices.Equal(strings.Fields(target.Query().Get("scope")), scopes(AuthDelegated)) {
		t.Fatal("OAuth consent did not request the shared delegated pool permissions")
	}
}

func TestEndpointSamplingCoverageAndIsolation(t *testing.T) {
	catalog := slices.Clone(Endpoints)
	for _, authType := range []AuthType{AuthDelegated, AuthApplication} {
		t.Run(string(authType), func(t *testing.T) {
			random := rand.New(rand.NewPCG(1, 2))
			pool := endpointsFor(authType)
			counts := map[int]bool{}
			first := map[string]bool{}
			covered := map[string]bool{}
			for range 1000 {
				sample := sampleEndpoints(authType, random)
				if len(sample) < 1 || len(sample) > len(pool) {
					t.Fatalf("sample size outside pool: %d", len(sample))
				}

				counts[len(sample)] = true
				first[sample[0].ID] = true
				seen := map[string]bool{}
				for _, ep := range sample {
					if ep.AuthType != authType || !slices.Contains(pool, ep) || seen[ep.ID] {
						t.Fatalf("duplicate or unsupported interface: %+v", ep)
					}

					seen[ep.ID], covered[ep.ID] = true, true
				}
			}

			if len(counts) != len(pool) || len(first) != len(pool) || len(covered) != len(pool) {
				t.Fatalf(
					"sampling never reached all sizes, interfaces, or first positions: %v %v %v",
					counts,
					covered,
					first,
				)
			}
		})
	}

	if !slices.Equal(catalog, Endpoints) {
		t.Fatal("sampling changed the shared catalog")
	}

	if sample := sampleEndpoints(AuthType("invalid"), rand.New(rand.NewPCG(1, 2))); len(sample) != 0 {
		t.Fatal("unknown authorization type received Graph interfaces")
	}
}

func TestManualAndScheduledRunsSampleSharedPool(t *testing.T) {
	for _, authType := range []AuthType{AuthDelegated, AuthApplication} {
		t.Run(string(authType), func(t *testing.T) {
			site := newTestSite(t)
			a := testAccount(t, site.db)
			a.AuthType, a.TenantID = authType, "example.onmicrosoft.com"
			mustNoError(t, site.db.Save(&a).Error)
			site.runner.random = rand.New(rand.NewPCG(1, 2))
			pool := endpointsFor(authType)
			requests := make(chan string, len(pool))
			site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host == "login.microsoftonline.com" {
					return jsonResponse(200, `{"access_token":"test-access","expires_in":3600}`), nil
				}

				if r.URL.Host != "graph.microsoft.com" || r.Method != http.MethodGet {
					t.Error("execution sent a non-Graph or mutating request")
				}

				requests <- r.URL.RequestURI()

				return jsonResponse(200, `{}`), nil
			})
			counts := map[int]bool{}
			for i := range 16 {
				trigger := TriggerManual
				if i%2 == 1 {
					trigger = TriggerScheduled
					mustNoError(t, site.db.Model(&a).Updates(map[string]any{
						"enabled": true, "next_run_at": time.Now().UTC().Add(-time.Minute),
					}).Error)
				}

				run, err := site.runner.Trigger(a.ID, trigger)
				mustNoError(t, err)
				finished := waitRun(t, site, run.ID)
				if finished.Status != RunSuccess || finished.RequestCount < 1 || finished.RequestCount > len(pool) {
					t.Fatalf("execution outside the shared pool bounds: %+v", finished)
				}

				counts[finished.RequestCount] = true
				seen := map[string]bool{}
				for _, result := range finished.Results {
					index := slices.IndexFunc(pool, func(ep Endpoint) bool { return ep.ID == result.Endpoint })
					if index < 0 || seen[result.Endpoint] {
						t.Fatalf("unsupported or repeated recorded interface: %s", result.Endpoint)
					}

					seen[result.Endpoint] = true
					select {
					case path := <-requests:
						if path != pool[index].Path || result.HTTPStatus != 200 || result.Attempts != 1 {
							t.Fatalf("call record does not match request: %s %+v", path, result)
						}
					default:
						t.Fatal("recorded interface was not actually requested")
					}
				}

				if len(requests) != 0 {
					t.Fatal("Graph requests were not individually persisted")
				}
			}

			if len(counts) < 2 {
				t.Fatal("execution used a fixed interface count")
			}
		})
	}
}

func TestInvalidStoredAuthTypeCannotSucceedWithoutCalls(t *testing.T) {
	site := newTestSite(t)
	a := testAccount(t, site.db)
	mustNoError(t, site.db.Model(&a).Update("auth_type", "invalid").Error)
	site.microsoft.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		t.Error("invalid authorization type reached Microsoft")
		return jsonResponse(200, `{}`), nil
	})
	run, err := site.runner.Trigger(a.ID, TriggerManual)
	mustNoError(t, err)
	finished := waitRun(t, site, run.ID)
	if finished.Status != RunFailed || finished.ErrorCode != "auth_type_invalid" || finished.RequestCount != 0 {
		t.Fatalf("empty interface pool was counted as success: %+v", finished)
	}
}

func TestConsentFailuresRequireAuthorization(t *testing.T) {
	for _, code := range []string{"invalid_scope", "consent_required", "interaction_required"} {
		t.Run(code, func(t *testing.T) {
			site := newTestSite(t)
			a := testAccount(t, site.db)
			mustNoError(t, site.db.Model(&a).Updates(map[string]any{
				"enabled": true, "next_run_at": time.Now().UTC().Add(-time.Minute),
			}).Error)
			site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Host == "graph.microsoft.com" {
					t.Error("Graph ran without the required consent")
				}

				return jsonResponse(400, `{"error":"`+code+`","error_description":"sensitive-provider-detail"}`), nil
			})
			run, err := site.runner.Trigger(a.ID, TriggerScheduled)
			mustNoError(t, err)
			finished := waitRun(t, site, run.ID)

			var saved Account
			mustNoError(t, site.db.First(&saved, a.ID).Error)
			if finished.Status != RunFailed || finished.ErrorCode != "authorization_"+code ||
				strings.Contains(finished.Error, "sensitive-provider-detail") || finished.RequestCount != 0 ||
				saved.Enabled || saved.NextRunAt != nil || saved.LastErrorCode != finished.ErrorCode {
				t.Fatalf("missing consent did not stop the schedule safely: %+v %+v", finished, saved)
			}
		})
	}
}
