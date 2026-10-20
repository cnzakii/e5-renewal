package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

func stateHash(state string) string {
	hash := sha256.Sum256([]byte(state))
	return hex.EncodeToString(hash[:])
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var input struct {
		RedirectURI string `json:"redirect_uri"`
	}
	if !decode(w, r, &input) {
		return
	}

	if input.RedirectURI == "" {
		input.RedirectURI = s.config.PublicURL + "/api/oauth/callback"
	}

	u, err := url.Parse(input.RedirectURI)
	if err != nil || !validWebURL(u) || u.RawQuery != "" || u.Fragment != "" {
		fail(
			w,
			400,
			newError(
				"redirect_uri_invalid",
				"Use an HTTP or HTTPS redirect URI without a query, fragment, or embedded credentials.",
			),
		)

		return
	}

	var a Account
	if err := s.db.WithContext(r.Context()).First(&a, id).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	if a.AuthType != AuthDelegated {
		fail(
			w,
			400,
			newError("browser_authorization_unnecessary", "Application authorization does not require a browser."),
		)

		return
	}

	var admin Admin
	if err := s.db.WithContext(r.Context()).First(&admin, 1).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	// Bind the pending grant to both account credentials and administrator authentication.
	state, verifier := randomToken(), oauth2.GenerateVerifier()
	now := time.Now().UTC()
	err = s.db.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at <= ? OR account_id = ?", now, a.ID).Delete(&OAuthAttempt{}).Error; err != nil {
			return err
		}

		return tx.Create(&OAuthAttempt{
			StateHash:     stateHash(state),
			AccountID:     a.ID,
			Revision:      a.Revision,
			AdminRevision: admin.Revision,
			Verifier:      verifier,
			RedirectURI:   input.RedirectURI,
			ExpiresAt:     now.Add(5 * time.Minute),
		}).Error
	})
	if err != nil {
		databaseError(w, r, err)
		return
	}

	respond(w, 200, map[string]string{"url": s.microsoft.AuthorizeURL(a, input.RedirectURI, state, verifier)})
}

// Both callback modes consume the same PKCE-bound, single-use authorization attempt.
func (s *Server) completeAuthorization(ctx context.Context, state, code, redirectURI string, accountID uint) error {
	if state == "" || code == "" || len(state) > 256 || len(code) > 16384 {
		return newError("oauth_callback_invalid", "The callback URL must contain a valid code and state.")
	}

	var attempt OAuthAttempt
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Where("state_hash = ? AND expires_at > ?", stateHash(state), time.Now().UTC())
		if accountID != 0 {
			query = query.Where("account_id = ?", accountID)
		}

		if err := query.First(&attempt).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			return newError(
				"oauth_state_invalid",
				"The authorization link expired or was already used. Start authorization again.",
			)
		}

		if redirectURI != "" && redirectURI != attempt.RedirectURI {
			return newError(
				"oauth_callback_mismatch",
				"The callback URL does not match this authorization attempt's redirect URI.",
			)
		}

		result := tx.Where("state_hash = ?", attempt.StateHash).Delete(&OAuthAttempt{})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return newError("oauth_state_invalid", "The authorization link was already used.")
		}

		return nil
	})
	if err != nil {
		return err
	}

	var (
		a     Account
		admin Admin
	)
	if err := s.db.WithContext(ctx).First(&a, attempt.AccountID).Error; err != nil {
		return err
	}

	if a.Revision != attempt.Revision {
		return errConflict
	}

	if err := s.db.WithContext(ctx).First(&admin, 1).Error; err != nil {
		return err
	}

	if admin.Revision != attempt.AdminRevision {
		return newError("authentication_changed", "Authentication changed. Sign in again.")
	}

	token, err := s.microsoft.Exchange(ctx, a, attempt.RedirectURI, code, attempt.Verifier)
	if err != nil {
		return err
	}

	// Recheck both revisions after the network request before saving the grant.
	result := s.db.WithContext(ctx).Model(&Account{}).Where("id = ? AND revision = ?", a.ID, a.Revision).
		Where("EXISTS (?)", s.db.Model(&Admin{}).Select("1").Where("id = 1 AND revision = ?", attempt.AdminRevision)).
		Updates(map[string]any{
			"refresh_token":   token.RefreshToken,
			"last_error":      "",
			"last_error_code": "",
			"revision":        a.Revision + 1,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return errConflict
	}

	return nil
}

func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	outcome := "success"
	if err := s.completeAuthorization(
		r.Context(),
		r.URL.Query().Get("state"),
		r.URL.Query().Get("code"),
		"",
		0,
	); err != nil {
		outcome = "failed"
		if errorCode(err) == "internal_error" {
			logDatabaseError(r.Context(), "authorization storage operation failed", err)
		} else {
			slog.Warn(
				"authorization failed",
				"error_code",
				errorCode(err),
				"request_id",
				middleware.GetReqID(r.Context()),
			)
		}
	}

	http.Redirect(w, r, s.baseHref+"accounts?authorization="+outcome, http.StatusSeeOther)
}

func (s *Server) exchange(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var input struct {
		CallbackURL string `json:"callback_url"`
	}
	if !decode(w, r, &input) {
		return
	}

	u, err := url.Parse(input.CallbackURL)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		fail(w, 400, newError("oauth_callback_invalid", "Enter the full callback URL containing code and state."))
		return
	}

	query := u.Query()
	u.RawQuery, u.ForceQuery = "", false
	if err := s.completeAuthorization(r.Context(), query.Get("state"), query.Get("code"), u.String(), id); err != nil {
		var (
			coded *codedError
			auth  *authFailure
		)
		if errors.As(err, &coded) || errors.As(err, &auth) {
			fail(w, 400, err)
		} else {
			databaseError(w, r, err)
		}

		return
	}

	respond(w, 200, map[string]bool{"authorized": true})
}
