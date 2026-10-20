package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func mustNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdministratorInitialization(t *testing.T) {
	for _, test := range []struct{ name, initial string }{{"generated", ""}, {"configured", testPassword}} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			db, err := OpenDB(Config{DataDir: directory, Driver: "sqlite", DSN: filepath.Join(directory, "test.db")})
			mustNoError(t, err)
			sqlDB, err := db.DB()
			mustNoError(t, err)
			t.Cleanup(func() { mustNoError(t, sqlDB.Close()) })
			generated, err := Initialize(db, test.initial)
			mustNoError(t, err)
			password := test.initial
			if test.initial == "" {
				password = generated
			} else if generated != "" {
				t.Fatal("configured password was returned for logging")
			}

			var admin Admin
			mustNoError(t, db.First(&admin, 1).Error)
			matches, err := argon2id.ComparePasswordAndHash(password, admin.PasswordHash)
			mustNoError(t, err)
			if !matches || password == "" {
				t.Fatal("initial password does not match its stored hash")
			}

			if repeated, err := Initialize(db, "must-not-replace-existing"); err != nil || repeated != "" {
				t.Fatal("restart generated or exposed an administrator password")
			}

			var preserved Admin
			mustNoError(t, db.First(&preserved, 1).Error)
			if preserved.PasswordHash != admin.PasswordHash {
				t.Fatal("restart changed the administrator password")
			}

			files, err := os.ReadDir(directory)
			mustNoError(t, err)
			for _, file := range files {
				if file.Name() != "test.db" && file.Name() != "test.db-wal" && file.Name() != "test.db-shm" {
					t.Fatalf("initialization wrote an unexpected file: %s", file.Name())
				}
			}
		})
	}
}

func TestDeploymentConfiguration(t *testing.T) {
	t.Setenv("E5_DB_DRIVER", "sqlite")
	t.Setenv("E5_DB_DSN", "")
	t.Setenv("E5_DATA_DIR", t.TempDir())
	t.Setenv("E5_LOG_LEVEL", "INFO")
	t.Setenv("E5_LOG_FILE", "")
	for _, test := range []struct {
		origin, format string
		valid          bool
	}{
		{"http://e5.example.com:8080", "text", true},
		{"http://192.168.1.10:8080", "json", true},
		{"https://e5.example.com", "text", true},
		{"http://127.0.0.1:8080", "", true},
		{"ftp://e5.example.com", "text", false},
		{"http://user:password@e5.example.com", "text", false},
		{"http://e5.example.com/admin", "text", true},
		{"http://e5.example.com?key=value", "text", false},
		{"http://e5.example.com", "unknown", false},
	} {
		t.Setenv("E5_PUBLIC_URL", test.origin)
		t.Setenv("E5_LOG_FORMAT", test.format)
		config, err := LoadConfig()
		if (err == nil) != test.valid {
			t.Fatalf("origin %s format %s: %v", test.origin, test.format, err)
		}

		if test.valid && test.format == "" && config.LogFormat != "text" {
			t.Fatal("default log format is not readable text")
		}
	}
}

func TestSchedulerDrainsDueAccountsSerially(t *testing.T) {
	site := newTestSite(t)
	entered, release := make(chan struct{}, 3), make(chan struct{}, 3)
	site.microsoft.Client.Transport = transportFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host == "login.microsoftonline.com" {
			entered <- struct{}{}
			select {
			case <-release:
			case <-request.Context().Done():
				return nil, request.Context().Err()
			}

			return jsonResponse(200, `{"access_token":"fixture-token","expires_in":3600}`), nil
		}

		return jsonResponse(200, `{}`), nil
	})
	for i := range 3 {
		account := testAccount(t, site.db)
		due := time.Now().UTC().Add(-time.Hour + time.Duration(i)*time.Minute)
		mustNoError(t, site.db.Model(&account).Updates(map[string]any{"enabled": true, "next_run_at": due}).Error)
	}

	for i := range 3 {
		site.runner.Tick()
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("due account did not start")
		}

		site.runner.Tick()

		var count int64
		mustNoError(t, site.db.Model(&Run{}).Count(&count).Error)
		if count != int64(i+1) {
			t.Fatal("another task started while the execution slot was occupied")
		}

		var run Run
		mustNoError(t, site.db.Order("id DESC").First(&run).Error)
		if run.AccountID != uint(i+1) || run.Trigger != TriggerScheduled {
			t.Fatal("due accounts were skipped or executed out of order")
		}

		release <- struct{}{}
		if waitRun(t, site, run.ID).Status != RunSuccess {
			t.Fatal("scheduled account failed")
		}
	}

	site.runner.Tick()

	var count int64
	mustNoError(t, site.db.Model(&Run{}).Count(&count).Error)
	if count != 3 {
		t.Fatal("completed due accounts were scheduled again immediately")
	}
}

func TestSchedulerStartsBothDueWorkAndRetention(t *testing.T) {
	site := newTestSite(t)
	account := testAccount(t, site.db)
	past := time.Now().UTC().Add(-31 * 24 * time.Hour)
	mustNoError(t, site.db.Model(&account).Updates(map[string]any{"enabled": true, "next_run_at": past}).Error)
	old := Run{AccountID: account.ID, Status: RunSuccess, StartedAt: past}
	mustNoError(t, site.db.Create(&old).Error)
	mustNoError(t, site.runner.Start())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var remaining int64
		mustNoError(t, site.db.Model(&Run{}).Where("id = ?", old.ID).Count(&remaining).Error)

		var current Account
		mustNoError(t, site.db.First(&current, account.ID).Error)
		if remaining == 0 && current.LastRunAt != nil && !site.runner.Busy() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("startup skipped due work or retention until its next scheduled interval")
}

func TestHTTPDiagnosticsAndPanicRedaction(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	handler := observeHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/panic":
			panic("panic-secret-value")
		case "/encode":
			respond(w, http.StatusOK, make(chan int))
		default:
			databaseError(w, r, &mysqlDriver.MySQLError{Number: 1045, Message: "database-secret-value"})
		}
	}))
	for _, path := range []string{"/database", "/panic", "/encode"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path+"?code=query-secret-value", nil)
		request.Header.Set("Authorization", "Bearer header-secret-value")
		request.Header.Set("X-Request-ID", "untrusted-secret-value")
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s returned %d", path, response.Code)
		}

		var payload struct {
			RequestID string `json:"request_id"`
		}
		mustNoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
		if payload.RequestID != response.Header().Get("X-Request-ID") {
			t.Fatal("failure response cannot be correlated with its log records")
		}

		if response.Header().Get("X-Request-ID") == "" ||
			response.Header().Get("X-Request-ID") == "untrusted-secret-value" {
			t.Fatal("response has no server-owned correlation ID")
		}
	}

	db := testDB(t)
	mustNoError(t, db.Callback().Query().Before("gorm:query").Register("test:oauth-storage-failure", func(tx *gorm.DB) {
		tx.Error = &mysqlDriver.MySQLError{Number: 1045, Message: "database-secret-value"}
	}))
	server := &Server{db: db, baseHref: "/"}
	for _, callback := range []string{
		"/api/oauth/callback?code=query-secret-value&state=fixture-state",
		"/api/oauth/callback?error=access_denied&error_description=query-secret-value",
	} {
		response := httptest.NewRecorder()
		observeHTTP(
			http.HandlerFunc(server.callback),
		).ServeHTTP(response, httptest.NewRequest(http.MethodGet, callback, nil))
		if response.Code != http.StatusSeeOther ||
			response.Header().Get("Location") != "/accounts?authorization=failed" {
			t.Fatal("failed authorization did not return to the account interface")
		}
	}

	logs := output.String()
	if !strings.Contains(logs, "authorization storage operation failed") ||
		!strings.Contains(logs, "oauth_callback_invalid") {
		t.Fatal("authorization failures lost their diagnostic records")
	}

	for _, secret := range []string{"panic-secret-value", "database-secret-value", "query-secret-value", "header-secret-value", "untrusted-secret-value"} {
		if strings.Contains(logs, secret) {
			t.Fatal("diagnostic output contains a sensitive value")
		}
	}

	decoder := json.NewDecoder(strings.NewReader(logs))
	records := 0
	sawDatabaseCode := false
	for {
		var record map[string]any
		err := decoder.Decode(&record)
		if errors.Is(err, io.EOF) {
			break
		}

		mustNoError(t, err)
		if record["level"] == nil || record["msg"] == nil {
			t.Fatal("diagnostic output is not structured JSON")
		}

		if record["database_code"] == float64(1045) {
			sawDatabaseCode = true
		}

		records++
	}

	if !sawDatabaseCode {
		t.Fatal("diagnostics lost the database failure code")
	}

	if records < 3 {
		t.Fatal("expected correlated diagnostic records")
	}
}

func TestExecutionPanicReleasesSlotAndKeepsCompletedCalls(t *testing.T) {
	site := newTestSite(t)
	account := testAccount(t, site.db)
	// Select profile followed by mail to panic after a completed call.
	site.runner.random = rand.New(rand.NewPCG(4915, 1))
	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(200, `{"access_token":"test-token","expires_in":3600}`), nil
		}

		if strings.Contains(r.URL.Path, "messages") {
			panic("provider panic")
		}

		return jsonResponse(200, `{}`), nil
	})
	run, err := site.runner.Trigger(account.ID, TriggerManual)
	mustNoError(t, err)
	finished := waitRun(t, site, run.ID)
	if finished.Status != RunFailed || finished.ErrorCode != "execution_panicked" || finished.RequestCount != 1 {
		t.Fatalf("panic outcome: %+v", finished)
	}

	site.microsoft.Client.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "login.microsoftonline.com" {
			return jsonResponse(200, `{"access_token":"test-token","expires_in":3600}`), nil
		}

		return jsonResponse(200, `{}`), nil
	})
	next, err := site.runner.Trigger(account.ID, TriggerManual)
	mustNoError(t, err)
	if finished := waitRun(t, site, next.ID); finished.Status != RunSuccess {
		t.Fatal("worker did not recover after a panic")
	}
}

func TestSchemaIndexesAndForeignKeys(t *testing.T) {
	db := testDB(t)
	for _, query := range []struct {
		sql, index string
		args       []any
	}{
		{"EXPLAIN QUERY PLAN SELECT id FROM accounts WHERE enabled = 1 AND next_run_at <= ? ORDER BY next_run_at, id LIMIT 1", "idx_accounts_due", []any{time.Now().UTC()}},
		{"EXPLAIN QUERY PLAN SELECT id FROM runs WHERE account_id = ? AND started_at >= ? ORDER BY started_at DESC, id DESC LIMIT 25", "idx_runs_account_started", []any{1, time.Now().UTC()}},
	} {
		var plans []struct{ Detail string }
		mustNoError(t, db.Raw(query.sql, query.args...).Scan(&plans).Error)
		used := false
		for _, plan := range plans {
			if strings.Contains(plan.Detail, query.index) {
				used = true
			}
		}

		if !used {
			t.Fatalf("query did not use %s: %+v", query.index, plans)
		}
	}

	if err := db.Create(&GraphCall{RunID: 9999, Endpoint: "profile", StartedAt: time.Now().UTC()}).Error; err == nil {
		t.Fatal("orphan Graph call was accepted")
	}

	sqlDB, err := db.DB()
	mustNoError(t, err)
	// Drop idle connections to exercise the same setup used after pool expiry.
	sqlDB.SetMaxIdleConns(0)
	if err := db.Create(&GraphCall{RunID: 9999, Endpoint: "profile", StartedAt: time.Now().UTC()}).Error; err == nil {
		t.Fatal("replacement connection accepted an orphan Graph call")
	}
}

func TestMicrosoftConnectionConcurrencyIsBounded(t *testing.T) {
	var active, peak atomic.Int32
	entered := make(chan struct{}, 8)
	release := make(chan struct{})

	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); current > old && !peak.CompareAndSwap(old, current); old = peak.Load() {
		}

		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	client := NewMicrosoft().Client

	var workers sync.WaitGroup
	defer func() {
		once.Do(func() { close(release) })
		workers.Wait()
		client.CloseIdleConnections()
		server.Close()
	}()
	workers.Add(8)
	for range 8 {
		go func() {
			defer workers.Done()
			response, err := client.Get(server.URL)
			if err != nil {
				t.Error(err)
				return
			}

			if err := response.Body.Close(); err != nil {
				t.Error(err)
			}
		}()
	}

	for range 4 {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("expected four connections to enter")
		}
	}

	select {
	case <-entered:
		t.Error("more than four connections entered concurrently")
	case <-time.After(50 * time.Millisecond):
	}

	once.Do(func() { close(release) })
	workers.Wait()
	if peak.Load() > 4 {
		t.Fatalf("unbounded connections: %d", peak.Load())
	}
}

func BenchmarkAccountOverview(b *testing.B) {
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
	b.Cleanup(func() { slog.SetDefault(previous) })
	for _, size := range []int{25, 500} {
		b.Run(strconv.Itoa(size)+"_accounts", func(b *testing.B) {
			site := newTestSite(b)
			accounts := make([]Account, size)
			for i := range accounts {
				accounts[i] = Account{
					Name:         "Benchmark account",
					AuthType:     AuthApplication,
					TenantID:     "example.com",
					ClientID:     testClientID,
					ClientSecret: "benchmark-fixture",
					MinMinutes:   1,
					MaxMinutes:   2,
					Revision:     1,
				}
			}

			mustNoError(b, site.db.CreateInBatches(&accounts, 100).Error)
			now := time.Now().UTC()
			runs := make([]Run, 0, size*7)
			for _, account := range accounts {
				for day := range 7 {
					started := now.AddDate(0, 0, -day)
					runs = append(
						runs,
						Run{
							AccountID:   account.ID,
							AccountName: account.Name,
							Status:      RunSuccess,
							Trigger:     TriggerScheduled,
							StartedAt:   started,
							Results:     []GraphCall{{Endpoint: "users", HTTPStatus: 200, StartedAt: started}},
						},
					)
				}
			}

			mustNoError(b, site.db.CreateInBatches(&runs, 100).Error)
			client := newBrowser(b, site)
			client.login(b)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				status, body := client.call(b, http.MethodGet, "/api/accounts", nil, "")
				if status != http.StatusOK {
					b.Fatalf("overview status %d", status)
				}

				b.ReportMetric(float64(len(body)), "response_B")
			}
		})
	}
}

type brokenResponseBody struct{}

func (brokenResponseBody) Read([]byte) (int, error) {
	return 0, errors.New("sensitive transport detail")
}
func (brokenResponseBody) Close() error { return nil }

func TestGraphReadFailureIsNotCountedAsSuccess(t *testing.T) {
	microsoft := NewMicrosoft()
	microsoft.Client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: brokenResponseBody{}, Header: make(http.Header)}, nil
	})
	result := microsoft.Call(context.Background(), "fixture-only-token", Endpoints[0])
	if result.ErrorCode != "graph_unavailable" || strings.Contains(result.Error, "sensitive") {
		t.Fatalf("incomplete response was accepted: %+v", result)
	}
}

func FuzzAccountValidation(f *testing.F) {
	f.Add("Personal", "delegated", 120, 240, "fixture-token", true)
	f.Add("Directory", "application", 1, 2, "", true)
	f.Add("Invalid", "unknown", -1, 0, "", false)
	f.Fuzz(func(t *testing.T, name, auth string, minimum, maximum int, token string, enabled bool) {
		if len(name) > 65536 || len(token) > 65536 {
			t.Skip()
		}

		secret := "fixture-only-secret"
		input := accountInput{
			Name:         name,
			AuthType:     AuthType(auth),
			ClientID:     testClientID,
			TenantID:     "example.com",
			ClientSecret: &secret,
			RefreshToken: &token,
			MinMinutes:   minimum,
			MaxMinutes:   maximum,
			Enabled:      enabled,
		}

		var account Account
		if input.apply(&account) != nil {
			return
		}

		if !account.AuthType.Valid() || account.MinMinutes < 1 || account.MaxMinutes < account.MinMinutes ||
			account.MaxMinutes > 43200 ||
			(account.Enabled && account.AuthType == AuthDelegated && account.RefreshToken == "") {
			t.Fatal("accepted account cannot satisfy the execution contract")
		}

		data, err := json.Marshal(account)
		mustNoError(t, err)

		var fields map[string]json.RawMessage
		mustNoError(t, json.Unmarshal(data, &fields))
		if fields["client_secret"] != nil || fields["refresh_token"] != nil {
			t.Fatal("accepted account exposes stored credentials")
		}
	})
}

func TestPersistencePanicCannotTerminateWorkerProcess(t *testing.T) {
	site := newTestSite(t)
	account := testAccount(t, site.db)
	mustNoError(
		t,
		site.db.Callback().Update().Before("gorm:update").Register("test:panic-on-run-update", func(tx *gorm.DB) {
			if tx.Statement.Table == "runs" {
				panic("persistence failed")
			}
		}),
	)
	_, err := site.runner.Trigger(account.ID, TriggerManual)
	mustNoError(t, err)
	deadline := time.Now().Add(3 * time.Second)
	for site.runner.Busy() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	if site.runner.Busy() {
		t.Fatal("persistence panic kept the execution slot locked")
	}

	response, err := site.http.Client().Get(site.http.URL + "/healthz")
	mustNoError(t, err)
	defer func() { mustNoError(t, response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		t.Fatal("service unavailable after persistence panic")
	}
}
