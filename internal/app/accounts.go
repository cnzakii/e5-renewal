package app

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	errConflict   = newError("account_conflict", "The account changed. Refresh and try again.")
	tenantPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,254}$`)
)

type accountInput struct {
	Name         string   `json:"name"`
	AuthType     AuthType `json:"auth_type"`
	ClientID     string   `json:"client_id"`
	TenantID     string   `json:"tenant_id"`
	ClientSecret *string  `json:"client_secret"`
	RefreshToken *string  `json:"refresh_token"`
	Enabled      bool     `json:"enabled"`
	MinMinutes   int      `json:"min_minutes"`
	MaxMinutes   int      `json:"max_minutes"`
	Revision     uint     `json:"revision"`
}

func (in accountInput) apply(a *Account) error {
	if strings.TrimSpace(in.Name) == "" || utf8.RuneCountInString(in.Name) > 120 {
		return newError("account_name_invalid", "The account name must contain 1 to 120 characters.")
	}

	if !in.AuthType.Valid() {
		return newError("auth_type_invalid", "Invalid authorization type.")
	}

	if _, err := uuid.Parse(in.ClientID); err != nil {
		return newError("client_id_invalid", "Client ID must be a valid UUID.")
	}

	if !tenantPattern.MatchString(in.TenantID) || strings.Contains(in.TenantID, "..") {
		return newError("tenant_invalid", "Invalid tenant ID or tenant domain.")
	}

	if in.AuthType == AuthApplication &&
		(in.TenantID == "common" || in.TenantID == "organizations" || in.TenantID == "consumers") {
		return newError("application_tenant_required", "Application authorization requires a tenant ID or domain.")
	}

	if in.MinMinutes < 1 || in.MaxMinutes < in.MinMinutes || in.MaxMinutes > 43200 {
		return newError(
			"interval_invalid",
			"Intervals must be between 1 and 43200 minutes, with maximum not less than minimum.",
		)
	}

	// Missing secret fields preserve saved values; changed credentials invalidate the old grant.
	credentialsChanged := a.AuthType != in.AuthType || a.ClientID != in.ClientID || a.TenantID != in.TenantID
	if in.ClientSecret != nil {
		if len(*in.ClientSecret) > 4096 || strings.TrimSpace(*in.ClientSecret) == "" {
			return newError("client_secret_invalid", "Client Secret is required and must not exceed 4096 bytes.")
		}

		credentialsChanged = credentialsChanged || a.ClientSecret != *in.ClientSecret
		a.ClientSecret = *in.ClientSecret
	}

	if a.ClientSecret == "" {
		return newError("client_secret_required", "Client Secret is required.")
	}

	if credentialsChanged {
		a.RefreshToken = ""
	}

	if in.RefreshToken != nil {
		if len(*in.RefreshToken) > 16384 {
			return newError("refresh_token_invalid", "Refresh Token is too long.")
		}

		a.RefreshToken = strings.TrimSpace(*in.RefreshToken)
	}

	if in.AuthType == AuthApplication {
		a.RefreshToken = ""
	}

	if in.Enabled && in.AuthType == AuthDelegated && a.RefreshToken == "" {
		return newError("authorization_required", "Complete OAuth authorization before enabling the schedule.")
	}

	scheduleChanged := a.Enabled != in.Enabled || a.MinMinutes != in.MinMinutes || a.MaxMinutes != in.MaxMinutes
	a.Name = strings.TrimSpace(in.Name)
	a.AuthType = in.AuthType
	a.ClientID = in.ClientID
	a.TenantID = in.TenantID
	a.Enabled = in.Enabled
	a.MinMinutes = in.MinMinutes
	a.MaxMinutes = in.MaxMinutes
	if !a.Enabled {
		a.NextRunAt = nil
	} else if scheduleChanged || a.NextRunAt == nil {
		a.NextRunAt = nextRun(*a, time.Now().UTC())
	}

	if credentialsChanged || in.RefreshToken != nil {
		a.LastError = ""
		a.LastErrorCode = ""
	}

	a.Revision++

	return nil
}

type accountResponse struct {
	Account
	HasClientSecret bool               `json:"has_client_secret"`
	HasRefreshToken bool               `json:"has_refresh_token"`
	Statistics      *AccountStatistics `json:"statistics,omitempty"`
}

func accountView(a Account) accountResponse {
	return accountResponse{Account: a, HasClientSecret: a.ClientSecret != "", HasRefreshToken: a.RefreshToken != ""}
}

func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	location, ok := requestLocation(w, r)
	if !ok {
		return
	}

	start := statisticsStart(time.Now(), 7, location)

	var accounts []Account
	if err := s.db.WithContext(r.Context()).Order("id").Find(&accounts).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	rows := make([]any, 0, len(accounts))
	statistics, err := s.statistics(r.Context(), 0, start)
	if err != nil {
		databaseError(w, r, err)
		return
	}

	for _, a := range accounts {
		view := accountView(a)
		value := fillStatistics(statistics[a.ID], start)
		view.Statistics = &value
		rows = append(rows, view)
	}

	respond(w, 200, map[string]any{"accounts": rows, "busy": s.runner.Busy()})
}

func (s *Server) account(w http.ResponseWriter, r *http.Request) {
	location, valid := requestLocation(w, r)
	if !valid {
		return
	}

	start := statisticsStart(time.Now(), 7, location)
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var a Account
	if err := s.db.WithContext(r.Context()).First(&a, id).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	statistics, err := s.statistics(r.Context(), a.ID, start)
	if err != nil {
		databaseError(w, r, err)
		return
	}

	view := accountView(a)
	value := fillStatistics(statistics[a.ID], start)
	view.Statistics = &value
	respond(w, 200, view)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var input accountInput
	if !decode(w, r, &input) {
		return
	}

	var a Account
	if err := input.apply(&a); err != nil {
		fail(w, 400, err)
		return
	}

	err := s.runner.Change(r.Context(), func(tx *gorm.DB) error { return tx.Create(&a).Error })
	if err != nil {
		databaseError(w, r, err)
		return
	}

	respond(w, 201, accountView(a))
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var input accountInput
	if !decode(w, r, &input) {
		return
	}

	var (
		a          Account
		validation error
	)
	err := s.runner.Change(r.Context(), func(tx *gorm.DB) error {
		if err := tx.First(&a, id).Error; err != nil {
			return err
		}

		if a.Revision != input.Revision {
			return errConflict
		}

		if err := input.apply(&a); err != nil {
			validation = err
			return err
		}

		return tx.Save(&a).Error
	})
	if validation != nil {
		fail(w, 400, validation)
		return
	}

	if err != nil {
		databaseError(w, r, err)
		return
	}

	respond(w, 200, accountView(a))
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	err := s.runner.Change(r.Context(), func(tx *gorm.DB) error {
		if err := tx.First(&Account{}, id).Error; err != nil {
			return err
		}

		if err := tx.Where("account_id = ?", id).Delete(&OAuthAttempt{}).Error; err != nil {
			return err
		}

		if err := tx.Where("run_id IN (?)", tx.Model(&Run{}).Select("id").Where("account_id = ?", id)).
			Delete(&GraphCall{}).
			Error; err != nil {
			return err
		}

		if err := tx.Where("account_id = ?", id).Delete(&Run{}).Error; err != nil {
			return err
		}

		return tx.Delete(&Account{}, id).Error
	})
	if err != nil {
		databaseError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) trigger(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	run, err := s.runner.Trigger(id, TriggerManual)
	if err != nil {
		databaseError(w, r, err)
		return
	}

	respond(w, 202, run)
}

func (s *Server) runs(w http.ResponseWriter, r *http.Request) {
	location, ok := requestLocation(w, r)
	if !ok {
		return
	}

	query := s.db.WithContext(r.Context()).Model(&Run{})
	if value := r.URL.Query().Get("account_id"); value != "" {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil || id == 0 {
			fail(w, 400, newError("account_id_invalid", "Invalid account ID."))
			return
		}

		query = query.Where("account_id = ?", id)
	}

	if value := r.URL.Query().Get("status"); value != "" {
		if !RunStatus(value).Valid() {
			fail(w, 400, newError("history_filter_invalid", "Invalid execution history filter."))
			return
		}

		query = query.Where("status = ?", value)
	}

	if value := r.URL.Query().Get("trigger"); value != "" {
		if !RunTrigger(value).Valid() {
			fail(w, 400, newError("history_filter_invalid", "Invalid execution history filter."))
			return
		}

		query = query.Where(map[string]any{"trigger": value})
	}

	days := 30
	if value := r.URL.Query().Get("days"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || !slices.Contains([]int{1, 7, 30}, parsed) {
			fail(w, 400, newError("history_filter_invalid", "Invalid execution history filter."))
			return
		}

		days = parsed
	}

	if value := r.URL.Query().Get("date"); value != "" {
		start, err := time.ParseInLocation("2006-01-02", value, location)
		if err != nil || len(value) != 10 {
			fail(w, 400, newError("history_filter_invalid", "Invalid execution history date."))
			return
		}

		query = query.Where(
			"id IN (?)",
			s.db.Model(&GraphCall{}).
				Select("run_id").
				Where("started_at >= ? AND started_at < ?", start.UTC(), start.AddDate(0, 0, 1).UTC()),
		)
	} else {
		query = query.Where("started_at >= ?", statisticsStart(time.Now(), days, location).UTC())
	}

	page := 1
	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 1000000 {
			fail(w, 400, newError("history_filter_invalid", "Invalid execution history filter."))
			return
		}

		page = parsed
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		databaseError(w, r, err)
		return
	}

	const pageSize = 25
	lastPage := max(1, (int(total)+pageSize-1)/pageSize)
	page = min(page, lastPage)

	var runs []Run
	if err := query.Preload("Results", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).
		Order("started_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&runs).
		Error; err != nil {
		databaseError(w, r, err)
		return
	}

	for i := range runs {
		runs[i].measure()
		if runs[i].Results == nil {
			runs[i].Results = []GraphCall{}
		}
	}

	respond(
		w,
		200,
		map[string]any{"runs": runs, "busy": s.runner.Busy(), "total": total, "page": page, "page_size": pageSize},
	)
}

func (s *Server) info(w http.ResponseWriter, r *http.Request) {
	respond(w, 200, map[string]string{"callback_url": s.config.PublicURL + "/api/oauth/callback"})
}
