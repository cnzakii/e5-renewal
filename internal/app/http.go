package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

type Server struct {
	db            *gorm.DB
	runner        *Runner
	microsoft     *Microsoft
	config        Config
	session       *scs.SessionManager
	loginLimit    *rate.Limiter
	assets        fs.FS
	indexHTML     []byte
	basePath      string
	baseHref      string
	origin        string
	authMu        sync.RWMutex
	adminRevision uint
}

// NewServer binds sessions and frontend assets to the configured public URL.
func NewServer(c Config, db *gorm.DB, runner *Runner, microsoft *Microsoft, assets fs.FS) (*Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin Admin
	if err := db.WithContext(ctx).Select("revision").First(&admin, 1).Error; err != nil {
		return nil, fmt.Errorf("load administrator state: %w", err)
	}

	publicURL, err := parsePublicURL(c.PublicURL)
	if err != nil {
		return nil, err
	}

	c.PublicURL = publicURL.String()
	baseHref := publicURL.EscapedPath() + "/"

	// Render the base once; the bundled assets are shared by every deployment path.
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return nil, fmt.Errorf("read frontend entry: %w", err)
	}

	const baseElement = `<base href="/" />`
	if strings.Count(string(index), baseElement) != 1 {
		return nil, errors.New("frontend entry must contain one base element")
	}

	index = []byte(strings.Replace(string(index), baseElement, `<base href="`+html.EscapeString(baseHref)+`" />`, 1))

	session := scs.New()
	session.Lifetime = 24 * time.Hour
	session.IdleTimeout = 4 * time.Hour
	session.Cookie.Name = "e5_session"
	session.Cookie.Path = baseHref
	session.Cookie.HttpOnly = true
	session.Cookie.Secure = strings.HasPrefix(c.PublicURL, "https://")
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Persist = false

	return &Server{
		config:        c,
		db:            db,
		runner:        runner,
		microsoft:     microsoft,
		assets:        assets,
		indexHTML:     index,
		basePath:      publicURL.Path,
		baseHref:      baseHref,
		origin:        publicURL.Scheme + "://" + publicURL.Host,
		session:       session,
		loginLimit:    rate.NewLimiter(rate.Every(12*time.Second), 5),
		adminRevision: admin.Revision,
	}, nil
}

func respond(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error(
			"response encoding failed",
			"error_type",
			fmt.Sprintf("%T", err),
			"request_id",
			w.Header().Get("X-Request-ID"),
		)
		status = http.StatusInternalServerError
		// A string-only fallback cannot fail JSON encoding.
		body, _ = json.Marshal(
			map[string]string{
				"code":       "internal_error",
				"error":      "Could not encode the response.",
				"request_id": w.Header().Get("X-Request-ID"),
			},
		)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(append(body, '\n')); err != nil {
		slog.Debug(
			"response write failed",
			"error_type",
			fmt.Sprintf("%T", err),
			"request_id",
			w.Header().Get("X-Request-ID"),
		)
	}
}

func fail(w http.ResponseWriter, status int, err error) {
	respond(
		w,
		status,
		map[string]string{"code": errorCode(err), "error": err.Error(), "request_id": w.Header().Get("X-Request-ID")},
	)
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		fail(w, 400, newError("request_invalid", "Invalid request body."))
		return false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		fail(w, 400, newError("request_json_invalid", "The request must contain one JSON object."))
		return false
	}

	return true
}

func parseID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil || id == 0 {
		fail(w, 400, newError("account_id_invalid", "Invalid account ID."))
		return 0, false
	}

	return uint(id), true
}

func databaseError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		fail(w, 404, newError("not_found", "Record not found."))
	case errors.Is(err, ErrBusy):
		fail(w, 409, err)
	case errors.Is(err, ErrClosed):
		fail(w, 503, err)
	case errors.Is(err, errConflict):
		fail(w, 409, err)
	default:
		logDatabaseError(r.Context(), "storage operation failed", err)
		fail(w, 500, newError("storage_failed", "The data operation failed."))
	}
}

func (s *Server) authenticated(ctx context.Context) bool {
	revision := s.session.GetInt(ctx, "admin_revision")
	if revision <= 0 {
		return false
	}

	s.authMu.RLock()
	defer s.authMu.RUnlock()

	return uint(revision) == s.adminRevision
}

func (s *Server) protected(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r.Context()) {
			fail(w, 401, newError("authentication_required", "Authentication required."))
			return
		}

		next(w, r)
	}
}

// Handler serves the interface and authenticated management API under one base path.
func (s *Server) Handler() http.Handler {
	mux := chi.NewRouter()
	mux.Use(middleware.GetHead, middleware.Timeout(40*time.Second))
	mux.MethodFunc("GET", "/healthz", func(w http.ResponseWriter, r *http.Request) {
		db, err := s.db.DB()
		if err != nil || db.PingContext(r.Context()) != nil {
			fail(w, 503, newError("database_unavailable", "The database is unavailable."))
			return
		}

		respond(w, 200, map[string]string{"status": "ok"})
	})
	mux.MethodFunc("GET", "/api/session", s.sessionInfo)
	mux.MethodFunc("POST", "/api/login", s.login)
	mux.MethodFunc("POST", "/api/logout", s.protected(s.logout))
	mux.MethodFunc("PUT", "/api/password", s.protected(s.password))
	mux.MethodFunc("GET", "/api/accounts", s.protected(s.accounts))
	mux.MethodFunc("POST", "/api/accounts", s.protected(s.createAccount))
	mux.MethodFunc("GET", "/api/accounts/{id}", s.protected(s.account))
	mux.MethodFunc("PUT", "/api/accounts/{id}", s.protected(s.updateAccount))
	mux.MethodFunc("DELETE", "/api/accounts/{id}", s.protected(s.deleteAccount))
	mux.MethodFunc("POST", "/api/accounts/{id}/run", s.protected(s.trigger))
	mux.MethodFunc("POST", "/api/accounts/{id}/authorize", s.protected(s.authorize))
	mux.MethodFunc("POST", "/api/accounts/{id}/exchange", s.protected(s.exchange))
	mux.MethodFunc("GET", "/api/oauth/callback", s.callback)
	mux.MethodFunc("GET", "/api/runs", s.protected(s.runs))
	mux.MethodFunc("GET", "/api/info", s.protected(s.info))
	mux.HandleFunc("/api/*", func(w http.ResponseWriter, _ *http.Request) {
		fail(w, 404, newError("route_not_found", "API endpoint not found."))
	})
	mux.Get("/", s.page)
	mux.Get("/accounts/{id:[0-9]+}", s.page)
	mux.Get("/*", s.page)

	handler := s.session.LoadAndSave(s.checkRequests(mux))

	return observeHTTP(s.withBasePath(handler))
}

func (s *Server) checkRequests(next http.Handler) http.Handler {
	const contentSecurityPolicy = "default-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)

		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				// Browser origins contain the scheme and host, never the deployment path.
				origin := r.Header.Get("Origin")
				if r.URL.Path == "/api/login" {
					if origin != s.origin {
						fail(w, 403, newError("origin_untrusted", "The request origin is not trusted."))
						return
					}

					next.ServeHTTP(w, r)

					return
				}

				if origin != "" && origin != s.origin {
					fail(w, 403, newError("origin_untrusted", "The request origin is not trusted."))
					return
				}

				expected := s.session.GetString(r.Context(), "csrf")
				provided := r.Header.Get("X-CSRF-Token")
				if expected == "" {
					fail(w, 401, newError("authentication_required", "Authentication required."))
					return
				}

				if subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
					fail(w, 403, newError("csrf_invalid", "The page session expired. Reload the page."))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withBasePath(next http.Handler) http.Handler {
	if s.basePath == "" {
		return next
	}

	prefixed := http.StripPrefix(s.basePath, next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == s.basePath:
			target := s.baseHref
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}

			http.Redirect(w, r, target, http.StatusPermanentRedirect)

		case strings.HasPrefix(r.URL.Path, s.basePath+"/"):
			// Match a complete path segment so /e5-other cannot enter /e5.
			prefixed.ServeHTTP(w, r)

		default:
			http.NotFound(w, r)
		}
	})
}

func (s *Server) sessionInfo(w http.ResponseWriter, r *http.Request) {
	authenticated := s.authenticated(r.Context())
	respond(w, 200, map[string]any{"authenticated": authenticated, "csrf": s.session.GetString(r.Context(), "csrf")})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.loginLimit.Allow() {
		w.Header().Set("Retry-After", "12")
		fail(w, 429, newError("login_rate_limited", "Too many login attempts. Try again shortly."))
		return
	}

	var input struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &input) {
		return
	}

	if len(input.Password) > 1024 {
		fail(w, 401, newError("password_incorrect", "Incorrect administrator password."))
		return
	}

	var admin Admin
	if err := s.db.WithContext(r.Context()).First(&admin, 1).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	ok, err := argon2id.ComparePasswordAndHash(input.Password, admin.PasswordHash)
	if err != nil {
		fail(w, 500, newError("password_verification_failed", "Password verification failed."))
		return
	}

	if !ok {
		fail(w, 401, newError("password_incorrect", "Incorrect administrator password."))
		return
	}

	if err := s.session.RenewToken(r.Context()); err != nil {
		fail(w, 500, newError("session_create_failed", "Could not create the session."))
		return
	}

	s.session.Put(r.Context(), "admin_revision", int(admin.Revision))
	s.session.Put(r.Context(), "csrf", randomToken())
	respond(w, 200, map[string]bool{"authenticated": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.session.Destroy(r.Context()); err != nil {
		fail(w, 500, newError("logout_failed", "Could not sign out."))
		return
	}

	respond(w, 200, map[string]bool{"authenticated": false})
}

func (s *Server) password(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !decode(w, r, &input) {
		return
	}

	if !s.loginLimit.Allow() {
		fail(w, 429, newError("rate_limited", "Too many requests."))
		return
	}

	if len(input.Current) > 1024 {
		fail(w, 400, newError("current_password_incorrect", "Incorrect current password."))
		return
	}

	var admin Admin
	if err := s.db.WithContext(r.Context()).First(&admin, 1).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	ok, err := argon2id.ComparePasswordAndHash(input.Current, admin.PasswordHash)
	if err != nil || !ok {
		fail(w, 400, newError("current_password_incorrect", "Incorrect current password."))
		return
	}

	hash, err := HashPassword(input.New)
	if err != nil {
		fail(w, 400, err)
		return
	}

	changed, err := s.updatePassword(r.Context(), admin.Revision, hash)
	if err != nil {
		databaseError(w, r, err)
		return
	}

	if !changed {
		fail(w, 409, newError("authentication_changed", "Authentication changed. Sign in again."))
		return
	}

	if err := s.session.Destroy(r.Context()); err != nil {
		fail(w, 500, newError("password_changed", "Password changed. Sign in again."))
		return
	}

	respond(w, 200, map[string]bool{"authenticated": false})
}

func (s *Server) updatePassword(ctx context.Context, revision uint, hash string) (bool, error) {
	// Serialize persistence and cache updates so concurrent changes cannot publish an older revision.
	s.authMu.Lock()
	defer s.authMu.Unlock()
	result := s.db.WithContext(ctx).
		Model(&Admin{}).
		Where("id = 1 AND revision = ?", revision).
		Updates(map[string]any{
			"password_hash": hash, "revision": revision + 1,
		})
	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected != 1 {
		return false, nil
	}

	s.adminRevision = revision + 1

	return true, nil
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	pagePath := r.URL.Path
	if chi.URLParam(r, "id") != "" {
		pagePath = "/accounts"
	}

	switch pagePath {
	case "/", "/login", "/accounts", "/runs", "/settings":
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write(s.indexHTML); err != nil {
			slog.Debug("page write failed", "request_id", middleware.GetReqID(r.Context()))
		}
	default:
		if !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.FileServer(http.FS(s.assets)).ServeHTTP(w, r)
	}
}
