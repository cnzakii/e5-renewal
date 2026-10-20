package app

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestSessionValidationDoesNotQueryAdministrator(t *testing.T) {
	site := newTestSite(t)
	client := newBrowser(t, site)
	client.login(t)

	var reads atomic.Int32
	mustNoError(
		t,
		site.db.Callback().Query().Before("gorm:query").Register("test:count-admin-reads", func(tx *gorm.DB) {
			if tx.Statement.Table == "admins" {
				reads.Add(1)
			}
		}),
	)
	for range 3 {
		for _, path := range []string{"/api/session", "/api/accounts", "/api/runs", "/api/info"} {
			if status, _ := client.call(t, http.MethodGet, path, nil, ""); status != http.StatusOK {
				t.Fatalf("authenticated %s returned %d", path, status)
			}
		}
	}

	if status, _ := client.call(t, http.MethodPost, "/api/logout", nil, ""); status != http.StatusOK {
		t.Fatalf("logout returned %d", status)
	}

	if status, _ := client.call(t, http.MethodGet, "/api/accounts", nil, ""); status != http.StatusUnauthorized {
		t.Fatal("logged-out session was accepted")
	}

	status, body := client.call(t, http.MethodGet, "/api/session", nil, "")

	var session struct{ Authenticated bool }
	mustNoError(t, json.Unmarshal(body, &session))
	if status != http.StatusOK || session.Authenticated || reads.Load() != 0 {
		t.Fatalf("session validation used SQL or retained authentication: reads=%d", reads.Load())
	}
}

func TestPasswordWriteFailurePreservesCachedAuthentication(t *testing.T) {
	site := newTestSite(t)
	client := newBrowser(t, site)
	client.login(t)
	mustNoError(
		t,
		site.db.Callback().Update().Before("gorm:update").Register("test:reject-password-write", func(tx *gorm.DB) {
			if tx.Statement.Table == "admins" {
				tx.Error = gorm.ErrInvalidDB
			}
		}),
	)
	status, _ := client.call(t, http.MethodPut, "/api/password", map[string]string{
		"current_password": testPassword,
		"new_password":     "replacement-password-not-saved",
	}, "")
	if status != http.StatusInternalServerError {
		t.Fatalf("failed password write returned %d", status)
	}

	if status, _ := client.call(t, http.MethodGet, "/api/accounts", nil, ""); status != http.StatusOK {
		t.Fatal("failed persistence invalidated the existing session")
	}

	client.login(t)
}

func TestConcurrentPasswordWritesKeepCacheConsistent(t *testing.T) {
	site := newTestSite(t)
	client := newBrowser(t, site)
	client.login(t)
	passwords := []string{"first-concurrent-password", "second-concurrent-password"}
	hashes := make([]string, len(passwords))
	for i, password := range passwords {
		hash, err := HashPassword(password)
		mustNoError(t, err)
		hashes[i] = hash
	}

	type outcome struct {
		index   int
		changed bool
		err     error
	}
	results := make(chan outcome, len(passwords))
	start := make(chan struct{})
	for i, hash := range hashes {
		go func() {
			<-start
			changed, err := site.server.updatePassword(context.Background(), 1, hash)
			results <- outcome{i, changed, err}
		}()
	}

	close(start)
	winner := -1
	for range passwords {
		var result outcome
		select {
		case result = <-results:
		case <-time.After(3 * time.Second):
			t.Fatal("concurrent password changes did not complete")
		}

		mustNoError(t, result.err)
		if result.changed {
			if winner != -1 {
				t.Fatal("both writes replaced the same password revision")
			}

			winner = result.index
		}
	}

	if winner == -1 {
		t.Fatal("neither password change succeeded")
	}

	if status, _ := client.call(t, http.MethodGet, "/api/accounts", nil, ""); status != http.StatusUnauthorized {
		t.Fatal("password change did not revoke the old session")
	}

	if status, _ := client.call(
		t,
		http.MethodPost,
		"/api/login",
		map[string]string{"password": passwords[winner]},
		"",
	); status != http.StatusOK {
		t.Fatal("persisted password could not sign in")
	}

	if status, _ := client.call(t, http.MethodGet, "/api/accounts", nil, ""); status != http.StatusOK {
		t.Fatal("cached revision disagrees with the persisted password")
	}
}

func TestServerRequiresInitializedAdministrator(t *testing.T) {
	db := testDB(t)
	mustNoError(t, db.Delete(&Admin{}, 1).Error)
	server, err := NewServer(Config{}, db, nil, nil, nil)
	if err == nil || server != nil {
		t.Fatal("server accepted an unavailable administrator revision")
	}
}

func BenchmarkSessionValidation(b *testing.B) {
	site := newTestSite(b)
	client := newBrowser(b, site)
	client.login(b)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		status, _ := client.call(b, http.MethodGet, "/api/session", nil, "")
		if status != http.StatusOK {
			b.Fatalf("session validation returned %d", status)
		}
	}
}
