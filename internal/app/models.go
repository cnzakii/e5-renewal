package app

import "time"

// AuthType selects the Microsoft OAuth grant used by an account.
type AuthType string

const (
	AuthDelegated   AuthType = "delegated"
	AuthApplication AuthType = "application"
)

func (a AuthType) Valid() bool { return a == AuthDelegated || a == AuthApplication }

// RunStatus describes the outcome of an execution, independently of its HTTP calls.
type RunStatus string

const (
	RunRunning     RunStatus = "running"
	RunSuccess     RunStatus = "success"
	RunPartial     RunStatus = "partial"
	RunFailed      RunStatus = "failed"
	RunInterrupted RunStatus = "interrupted"
)

func (s RunStatus) Valid() bool {
	switch s {
	case RunRunning, RunSuccess, RunPartial, RunFailed, RunInterrupted:
		return true
	}

	return false
}

// RunTrigger records why an execution was started.
type RunTrigger string

const (
	TriggerManual    RunTrigger = "manual"
	TriggerScheduled RunTrigger = "scheduled"
)

func (t RunTrigger) Valid() bool { return t == TriggerManual || t == TriggerScheduled }

type Admin struct {
	ID           uint   `gorm:"primaryKey"`
	PasswordHash string `gorm:"type:text;not null"`
	Revision     uint
}

type Account struct {
	ID            uint       `gorm:"primaryKey"                        json:"id"`
	Name          string     `gorm:"size:120;not null"                 json:"name"`
	AuthType      AuthType   `gorm:"size:20;not null"                  json:"auth_type"`
	TenantID      string     `gorm:"size:255;not null"                 json:"tenant_id"`
	ClientID      string     `gorm:"size:64;not null"                  json:"client_id"`
	ClientSecret  string     `gorm:"type:text"                         json:"-"`
	RefreshToken  string     `gorm:"type:text"                         json:"-"`
	Enabled       bool       `gorm:"index:idx_accounts_due,priority:1" json:"enabled"`
	MinMinutes    int        `                                         json:"min_minutes"`
	MaxMinutes    int        `                                         json:"max_minutes"`
	NextRunAt     *time.Time `gorm:"index:idx_accounts_due,priority:2" json:"next_run_at"`
	LastRunAt     *time.Time `                                         json:"last_run_at"`
	LastError     string     `gorm:"type:text"                         json:"last_error"`
	LastErrorCode string     `gorm:"size:64"                           json:"last_error_code"`
	Revision      uint       `                                         json:"revision"`
	CreatedAt     time.Time  `                                         json:"created_at"`
	UpdatedAt     time.Time  `                                         json:"updated_at"`
}

type Run struct {
	ID          uint        `gorm:"primaryKey;index:idx_runs_account_started,priority:3,sort:desc" json:"id"`
	AccountID   uint        `gorm:"not null;index:idx_runs_account_started,priority:1"             json:"account_id"`
	AccountName string      `gorm:"size:120"                                                       json:"account_name"`
	Trigger     RunTrigger  `gorm:"size:20"                                                        json:"trigger"`
	Status      RunStatus   `gorm:"size:20;index"                                                  json:"status"`
	Error       string      `gorm:"type:text"                                                      json:"error"`
	ErrorCode   string      `gorm:"size:64"                                                        json:"error_code"`
	Results     []GraphCall `gorm:"foreignKey:RunID"                                               json:"results"`
	// Counts are derived from call records for API responses.
	RequestCount int        `gorm:"-"                                                         json:"request_count"`
	SuccessCount int        `gorm:"-"                                                         json:"success_count"`
	StartedAt    time.Time  `gorm:"index;index:idx_runs_account_started,priority:2,sort:desc" json:"started_at"`
	FinishedAt   *time.Time `                                                                 json:"finished_at"`
}

type OAuthAttempt struct {
	StateHash     string `gorm:"primaryKey;size:64"`
	AccountID     uint
	Revision      uint
	AdminRevision uint
	Verifier      string    `gorm:"type:text"`
	RedirectURI   string    `gorm:"type:text"`
	ExpiresAt     time.Time `gorm:"index"`
}

type GraphCall struct {
	ID         uint      `gorm:"primaryKey"       json:"id"`
	RunID      uint      `gorm:"index;not null"   json:"-"`
	StartedAt  time.Time `gorm:"index;not null"   json:"started_at"`
	DurationMS int64     `                        json:"duration_ms"`
	Attempts   int       `                        json:"attempts"`
	Endpoint   string    `gorm:"size:40;not null" json:"endpoint"`
	HTTPStatus int       `                        json:"http_status"`
	Error      string    `gorm:"type:text"        json:"error"`
	ErrorCode  string    `gorm:"size:64"          json:"error_code"`
}
