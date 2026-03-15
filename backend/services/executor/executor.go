package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/graph"
	"e5-renewal/backend/services/oauth"

	"github.com/google/uuid"
)

// delegatedScope builds the scope string for auth_code token requests.
func delegatedScope() string {
	return strings.Join(graph.DelegatedScopeURLs(), " ") + " offline_access"
}

// Executor encapsulates the full task execution flow:
// get token → pick endpoints → call Graph API → write TaskLog + EndpointLog.
type Executor struct {
	OAuth *oauth.Service
	Graph *graph.Caller
	Rand  *rand.Rand
}

// New creates an executor instance.
func New(oauthSvc *oauth.Service, rng *rand.Rand) *Executor {
	return &Executor{
		OAuth: oauthSvc,
		Graph: &graph.Caller{},
		Rand:  rng,
	}
}

// Run executes a full task for the given account:
// 1. Obtain access token (refresh or client_credentials)
// 2. Pick random endpoints based on auth_type
// 3. Call each endpoint via Graph API
// 4. Write TaskLog + EndpointLogs in a single transaction
func (e *Executor) Run(ctx context.Context, account models.Account, triggerType string) (*models.TaskLog, error) {
	var authInfo models.AuthInfoData
	if err := json.Unmarshal([]byte(account.AuthInfo), &authInfo); err != nil {
		return nil, fmt.Errorf("decode auth_info: %w", err)
	}

	// 1. Obtain access token
	accessToken, err := e.acquireToken(ctx, account, authInfo)
	if err != nil {
		taskLog := e.recordTokenFailure(ctx, account, triggerType, err)
		return taskLog, fmt.Errorf("acquire token: %w", err)
	}

	// 2. Pick endpoints: manual trigger calls all, scheduled picks random subset
	cfg := config.Get().Scheduler
	pool := graph.EndpointsForAuthType(account.AuthType)
	var endpoints []graph.Endpoint
	if triggerType == models.TriggerManual {
		endpoints = pool
	} else {
		endpoints = graph.PickRandomEndpoints(pool, cfg.EndpointsMin, cfg.EndpointsMax, e.Rand)
	}

	// 3. Call Graph API endpoints
	runResult := e.Graph.CallEndpoints(ctx, accessToken, endpoints, 1, 3)

	// 4. Write TaskLog + EndpointLogs
	taskLog := e.recordResult(ctx, account, triggerType, runResult)
	return taskLog, nil
}

// TriggerResult is the full result of a manual trigger.
type TriggerResult struct {
	TaskLog   *models.TaskLog      `json:"task_log"`
	Endpoints []models.EndpointLog `json:"endpoints"`
}

// RunManual triggers execution manually, calls all endpoints with no delay, and returns the result synchronously.
func (e *Executor) RunManual(ctx context.Context, account models.Account) (*TriggerResult, error) {
	var authInfo models.AuthInfoData
	if err := json.Unmarshal([]byte(account.AuthInfo), &authInfo); err != nil {
		return nil, fmt.Errorf("decode auth_info: %w", err)
	}

	accessToken, err := e.acquireToken(ctx, account, authInfo)
	if err != nil {
		taskLog := e.recordTokenFailure(ctx, account, models.TriggerManual, err)
		eps, _ := database.EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
		return &TriggerResult{TaskLog: taskLog, Endpoints: eps}, fmt.Errorf("acquire token: %w", err)
	}

	pool := graph.EndpointsForAuthType(account.AuthType)
	// Manual trigger: call all endpoints with zero delay.
	runResult := e.Graph.CallEndpoints(ctx, accessToken, pool, 0, 0)
	taskLog := e.recordResult(ctx, account, models.TriggerManual, runResult)
	eps, _ := database.EndpointLogs.ListByTaskLogID(ctx, taskLog.ID)
	return &TriggerResult{TaskLog: taskLog, Endpoints: eps}, nil
}

func (e *Executor) acquireToken(ctx context.Context, account models.Account, authInfo models.AuthInfoData) (string, error) {
	logger := slog.With("subsystem", "executor", "account_id", account.ID, "auth_type", account.AuthType)

	switch account.AuthType {
	case models.AuthTypeAuthCode:
		resp, err := e.OAuth.RefreshToken(ctx, authInfo.TenantID, authInfo.ClientID, authInfo.ClientSecret, authInfo.RefreshToken, delegatedScope())
		if err != nil {
			logger.Error("auth code token refresh failed", "error", err)
			return "", err
		}
		logger.Info("auth code token refresh succeeded", "refresh_token_rotated", resp.RefreshToken != "")
		if resp.RefreshToken != "" {
			authInfo.RefreshToken = resp.RefreshToken
			newAuthInfo, _ := json.Marshal(authInfo)
			if err := database.Accounts.UpdateAuthInfo(ctx, account.ID, string(newAuthInfo), account.AuthExpiresAt); err != nil {
				logger.Warn("failed to persist rotated refresh token", "error", err)
			}
		}
		return resp.AccessToken, nil

	case models.AuthTypeClientCredentials:
		resp, err := e.OAuth.AcquireClientToken(ctx, authInfo.TenantID, authInfo.ClientID, authInfo.ClientSecret)
		if err != nil {
			logger.Error("client credentials token acquisition failed", "error", err)
			return "", err
		}
		logger.Info("client credentials token acquisition succeeded")
		return resp.AccessToken, nil

	default:
		return "", fmt.Errorf("unknown auth_type: %s", account.AuthType)
	}
}

func (e *Executor) recordTokenFailure(ctx context.Context, account models.Account, triggerType string, tokenErr error) *models.TaskLog {
	now := time.Now().UTC()
	finishedAt := now
	taskLog := models.TaskLog{
		AccountID:      account.ID,
		RunID:          uuid.New().String(),
		TriggerType:    triggerType,
		TotalEndpoints: 0,
		SuccessCount:   0,
		FailCount:      1,
		StartedAt:      now,
		FinishedAt:     &finishedAt,
	}
	endpoints := []models.EndpointLog{{
		EndpointName: "token",
		HTTPStatus:   0,
		Success:      false,
		ErrorMessage: tokenErr.Error(),
		ExecutedAt:   now,
	}}
	if err := database.TaskLogs.CreateWithEndpoints(ctx, &taskLog, endpoints); err != nil {
		slog.Error("failed to record token failure", "subsystem", "executor", "account_id", account.ID, "trigger_type", triggerType, "error", err)
	}
	return &taskLog
}

func (e *Executor) recordResult(ctx context.Context, account models.Account, triggerType string, result graph.RunResult) *models.TaskLog {
	now := time.Now().UTC()
	finishedAt := now
	taskLog := models.TaskLog{
		AccountID:      account.ID,
		RunID:          uuid.New().String(),
		TriggerType:    triggerType,
		TotalEndpoints: len(result.Results),
		SuccessCount:   result.Succeeded,
		FailCount:      result.Failed,
		StartedAt:      now,
		FinishedAt:     &finishedAt,
	}
	if len(result.Results) > 0 {
		taskLog.StartedAt = result.Results[0].ExecutedAt
	}

	endpoints := make([]models.EndpointLog, 0, len(result.Results))
	for _, r := range result.Results {
		endpoints = append(endpoints, models.EndpointLog{
			EndpointName: r.Endpoint,
			Scope:        r.Scope,
			HTTPStatus:   r.Status,
			Success:      r.Error == "",
			ErrorMessage: r.Error,
			ResponseBody: r.ResponseBody,
			ExecutedAt:   r.ExecutedAt,
		})
	}

	if err := database.TaskLogs.CreateWithEndpoints(ctx, &taskLog, endpoints); err != nil {
		slog.Error("failed to record task result", "subsystem", "executor", "account_id", account.ID, "trigger_type", triggerType, "error", err)
	}
	return &taskLog
}
