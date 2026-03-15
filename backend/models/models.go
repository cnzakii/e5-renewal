package models

import "time"

// Auth type constants.
const (
	AuthTypeAuthCode          = "auth_code"
	AuthTypeClientCredentials = "client_credentials"
)

// Trigger type constants.
const (
	TriggerScheduled = "scheduled"
	TriggerManual    = "manual"
)

// Account represents an OAuth2 account for E5 subscription renewal.
// Auth credentials are stored as a JSON blob in AuthInfo.
type Account struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"size:120;uniqueIndex;not null" json:"name"`
	AuthType      string     `gorm:"size:50;not null" json:"auth_type"`
	AuthInfo      string     `gorm:"type:text;not null" json:"-"` // JSON: {client_id, client_secret, tenant_id, refresh_token?}
	NotifyEnabled bool       `json:"notify_enabled"`
	AuthExpiresAt *time.Time `json:"auth_expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AuthInfoData is the deserialized form of Account.AuthInfo.
type AuthInfoData struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	TenantID     string `json:"tenant_id"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// TaskLog represents a single execution of an account's scheduled renewal task.
// Each run may call multiple Graph API endpoints.
type TaskLog struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	AccountID      uint       `gorm:"index;not null" json:"account_id"`
	RunID          string     `gorm:"size:64;uniqueIndex" json:"run_id"`
	TriggerType    string     `gorm:"size:20;not null" json:"trigger_type"`
	TotalEndpoints int        `json:"total_endpoints"`
	SuccessCount   int        `json:"success_count"`
	FailCount      int        `json:"fail_count"`
	StartedAt      time.Time  `gorm:"index" json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EndpointLog represents a single API endpoint call within a TaskLog.
type EndpointLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	TaskLogID    uint      `gorm:"index;not null" json:"task_log_id"`
	EndpointName string    `gorm:"size:120" json:"endpoint_name"`
	Scope        string    `gorm:"size:120" json:"scope"`
	HTTPStatus   int       `json:"http_status"`
	Success      bool      `json:"success"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	ResponseBody string    `gorm:"type:text" json:"response_body"` // recorded on failure
	ExecutedAt   time.Time `json:"executed_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// Schedule holds per-account scheduling configuration and state.
type Schedule struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	AccountID      uint       `gorm:"uniqueIndex;not null" json:"account_id"`
	Enabled        bool       `json:"enabled"`
	PauseThreshold int        `json:"pause_threshold"` // health % below which auto-pause, 0 = disabled
	Paused         bool       `json:"paused"`
	PauseReason    string     `gorm:"size:255" json:"pause_reason"`
	NextRunAt      *time.Time `json:"next_run_at"`
	LastRunAt      *time.Time `json:"last_run_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Setting stores key-value configuration in the database.
type Setting struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:120;uniqueIndex;not null"`
	Value     string `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NotificationConfig is the deserialized form of the notification setting.
type NotificationConfig struct {
	URL              string `json:"url"`
	Language         string `json:"language"` // "zh" or "en", defaults to "zh"
	OnAuthExpiry     bool   `json:"on_auth_expiry"`
	ExpiryDaysBefore int    `json:"expiry_days_before"`
	OnTaskAllFailed  bool   `json:"on_task_all_failed"`
	OnHealthLow      bool   `json:"on_health_low"`
	HealthThreshold  int    `json:"health_threshold"`
}
