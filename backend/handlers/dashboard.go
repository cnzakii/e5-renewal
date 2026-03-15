package handlers

import (
	"context"
	"net/http"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/models"

	"github.com/gin-gonic/gin"
)

func RegisterDashboardRoutes(r *gin.Engine) {
	prefix := config.Get().Server.PathPrefix
	group := r.Group(prefix + "/api/dashboard")
	group.Use(middleware.RequireAuth())
	group.GET("/summary", dashboardSummaryHandler())
	group.GET("/trend", dashboardTrendHandler())
	group.GET("/account-health", dashboardAccountHealthHandler())
	group.GET("/recent-logs", dashboardRecentLogsHandler())
}

// --- Summary ---

type dashboardSummary struct {
	TotalAccounts    int64   `json:"total_accounts"`
	SuccessRate      float64 `json:"success_rate"`
	TotalRuns        int64   `json:"total_runs"`
	ErrorCount       int64   `json:"error_count"`
	AuthCodeCount    int64   `json:"auth_code_count"`
	CredentialsCount int64   `json:"credentials_count"`
}

func dashboardSummaryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		since := parsePeriod(c)

		totalAccounts, _ := database.Accounts.CountAll(ctx)
		authCodeCount, _ := database.Accounts.CountByAuthType(ctx, models.AuthTypeAuthCode)
		credentialsCount := totalAccounts - authCodeCount

		totalRuns, _ := database.TaskLogs.CountInPeriod(ctx, since)
		errorCount, _ := database.TaskLogs.CountErrorsInPeriod(ctx, since)

		var successRate float64
		if totalRuns > 0 {
			successRate = float64(totalRuns-errorCount) / float64(totalRuns) * 100
		}

		c.JSON(http.StatusOK, dashboardSummary{
			TotalAccounts:    totalAccounts,
			SuccessRate:      successRate,
			TotalRuns:        totalRuns,
			ErrorCount:       errorCount,
			AuthCodeCount:    authCodeCount,
			CredentialsCount: credentialsCount,
		})
	}
}

// --- Trend ---

type dashboardTrendItem struct {
	Date            string `json:"date"`
	Success         int    `json:"success"`
	Failure         int    `json:"failure"`
	AuthCodeRuns    int    `json:"auth_code_runs"`
	CredentialsRuns int    `json:"credentials_runs"`
}

func dashboardTrendHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		period := c.DefaultQuery("period", "7d")

		accounts, _ := database.Accounts.List(ctx)
		accountAuthType := make(map[uint]string, len(accounts))
		for _, a := range accounts {
			accountAuthType[a.ID] = a.AuthType
		}

		trend := buildTrend(ctx, accountAuthType, period, time.Now().UTC())
		c.JSON(http.StatusOK, trend)
	}
}

// --- Account Health ---

type dashboardAccountHealth struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	AuthType    string  `json:"auth_type"`
	Health      float64 `json:"health"`
	TotalRuns   int     `json:"total_runs"`
	SuccessRuns int     `json:"success_runs"`
	LastRun     string  `json:"last_run"`
}

func dashboardAccountHealthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		accounts, _ := database.Accounts.List(ctx)
		result := buildAccountHealth(ctx, accounts)
		c.JSON(http.StatusOK, result)
	}
}

// --- Recent Logs ---

type dashboardRecentLog struct {
	ID              uint   `json:"id"`
	AccountName     string `json:"account_name"`
	AccountAuthType string `json:"account_auth_type"`
	TriggerType     string `json:"trigger_type"`
	TotalEndpoints  int    `json:"total_endpoints"`
	SuccessCount    int    `json:"success_count"`
	FailCount       int    `json:"fail_count"`
	StartedAt       string `json:"started_at"`
	FinishedAt      string `json:"finished_at"`
}

func dashboardRecentLogsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		accounts, _ := database.Accounts.List(ctx)
		result := buildRecentLogs(ctx, accounts)
		c.JSON(http.StatusOK, result)
	}
}

// --- helpers ---

func parsePeriod(c *gin.Context) time.Time {
	period := c.DefaultQuery("period", "7d")
	now := time.Now().UTC()
	switch period {
	case "1d":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return time.Time{}
	}
}

func buildTrend(ctx context.Context, accountAuthType map[uint]string, period string, now time.Time) []dashboardTrendItem {
	type bucket struct {
		label string
		start time.Time
		end   time.Time
	}
	var buckets []bucket

	if period == "1d" {
		for i := 23; i >= 0; i-- {
			t := now.Add(-time.Duration(i) * time.Hour)
			start := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
			end := start.Add(time.Hour)
			buckets = append(buckets, bucket{start.Format("15:04"), start, end})
		}
	} else {
		days := 7
		switch period {
		case "30d":
			days = 30
		case "all":
			days = 90
		}
		for i := days - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			start := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 0, 1)
			buckets = append(buckets, bucket{start.Format("1-02"), start, end})
		}
	}

	result := make([]dashboardTrendItem, 0, len(buckets))
	for _, b := range buckets {
		logs, _ := database.TaskLogs.FindInTimeRange(ctx, b.start, b.end)
		item := dashboardTrendItem{Date: b.label}
		for i := range logs {
			if logs[i].FailCount == 0 {
				item.Success++
			} else {
				item.Failure++
			}
			if accountAuthType[logs[i].AccountID] == models.AuthTypeAuthCode {
				item.AuthCodeRuns++
			} else {
				item.CredentialsRuns++
			}
		}
		result = append(result, item)
	}
	return result
}

func buildAccountHealth(ctx context.Context, accounts []models.Account) []dashboardAccountHealth {
	result := make([]dashboardAccountHealth, 0, len(accounts))
	for _, acc := range accounts {
		healthVal := 0.0
		if h := computeHealth(ctx, acc.ID); h != nil {
			healthVal = *h
		}

		totalRuns, _ := database.TaskLogs.CountByAccount(ctx, acc.ID)
		successRuns, _ := database.TaskLogs.CountSuccessByAccount(ctx, acc.ID)

		lastRunStr := ""
		if last, err := database.TaskLogs.LastByAccount(ctx, acc.ID); err == nil {
			lastRunStr = last.StartedAt.Format(time.RFC3339)
		}

		result = append(result, dashboardAccountHealth{
			ID:          acc.ID,
			Name:        acc.Name,
			AuthType:    acc.AuthType,
			Health:      healthVal,
			TotalRuns:   int(totalRuns),
			SuccessRuns: int(successRuns),
			LastRun:     lastRunStr,
		})
	}
	return result
}

func buildRecentLogs(ctx context.Context, accounts []models.Account) []dashboardRecentLog {
	accountMap := make(map[uint][2]string)
	for _, a := range accounts {
		accountMap[a.ID] = [2]string{a.Name, a.AuthType}
	}

	logs, _ := database.TaskLogs.RecentN(ctx, 10)
	result := make([]dashboardRecentLog, 0, len(logs))
	for i := range logs {
		acc := accountMap[logs[i].AccountID]
		finishedAt := ""
		if logs[i].FinishedAt != nil {
			finishedAt = logs[i].FinishedAt.Format(time.RFC3339)
		}
		result = append(result, dashboardRecentLog{
			ID:              logs[i].ID,
			AccountName:     acc[0],
			AccountAuthType: acc[1],
			TriggerType:     logs[i].TriggerType,
			TotalEndpoints:  logs[i].TotalEndpoints,
			SuccessCount:    logs[i].SuccessCount,
			FailCount:       logs[i].FailCount,
			StartedAt:       logs[i].StartedAt.Format(time.RFC3339),
			FinishedAt:      finishedAt,
		})
	}
	return result
}
