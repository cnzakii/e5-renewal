package handlers

import (
	"net/http"
	"strconv"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterLogRoutes(r *gin.Engine) {
	prefix := config.Get().Server.PathPrefix
	group := r.Group(prefix + "/api")
	group.Use(middleware.RequireAuth())
	group.GET("/logs", listTaskLogsHandler())
	group.GET("/logs/:task_log_id/endpoints", listEndpointLogsHandler())
}

type taskLogRow struct {
	ID              uint    `json:"id"`
	AccountID       uint    `json:"account_id"`
	AccountName     string  `json:"account_name"`
	AccountAuthType string  `json:"account_auth_type"`
	RunID           string  `json:"run_id"`
	TriggerType     string  `json:"trigger_type"`
	TotalEndpoints  int     `json:"total_endpoints"`
	SuccessCount    int     `json:"success_count"`
	FailCount       int     `json:"fail_count"`
	StartedAt       string  `json:"started_at"`
	FinishedAt      *string `json:"finished_at"`
	CreatedAt       string  `json:"created_at"`
}

func listTaskLogsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		page, pageSize := parsePagination(c)

		filter := database.TaskLogFilter{
			Page:     page,
			PageSize: pageSize,
		}

		if v := c.Query("id"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 64); err == nil {
				filter.ID = uint(id)
			}
		}
		if v := c.Query("account_id"); v != "" {
			if id, err := strconv.ParseUint(v, 10, 64); err == nil {
				filter.AccountID = uint(id)
			}
		}
		filter.TriggerType = c.Query("trigger_type")
		filter.Status = c.Query("status")
		if v := c.Query("date_from"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				filter.DateFrom = &t
			}
		}
		if v := c.Query("date_to"); v != "" {
			if t, err := time.Parse("2006-01-02", v); err == nil {
				endOfDay := t.AddDate(0, 0, 1)
				filter.DateTo = &endOfDay
			}
		}

		logs, total, err := database.TaskLogs.ListWithTotal(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query logs"})
			return
		}

		accounts, _ := database.Accounts.List(ctx)
		accountMap := make(map[uint][2]string, len(accounts))
		for _, a := range accounts {
			accountMap[a.ID] = [2]string{a.Name, a.AuthType}
		}

		rows := make([]taskLogRow, 0, len(logs))
		for i := range logs {
			acc := accountMap[logs[i].AccountID]
			var finishedAt *string
			if logs[i].FinishedAt != nil {
				s := logs[i].FinishedAt.Format(time.RFC3339)
				finishedAt = &s
			}
			rows = append(rows, taskLogRow{
				ID:              logs[i].ID,
				AccountID:       logs[i].AccountID,
				AccountName:     acc[0],
				AccountAuthType: acc[1],
				RunID:           logs[i].RunID,
				TriggerType:     logs[i].TriggerType,
				TotalEndpoints:  logs[i].TotalEndpoints,
				SuccessCount:    logs[i].SuccessCount,
				FailCount:       logs[i].FailCount,
				StartedAt:       logs[i].StartedAt.Format(time.RFC3339),
				FinishedAt:      finishedAt,
				CreatedAt:       logs[i].CreatedAt.Format(time.RFC3339),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"items":     rows,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

func listEndpointLogsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		taskLogID, err := strconv.ParseUint(c.Param("task_log_id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task_log_id"})
			return
		}

		logs, err := database.EndpointLogs.ListByTaskLogID(c.Request.Context(), uint(taskLogID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query endpoint logs"})
			return
		}
		c.JSON(http.StatusOK, logs)
	}
}

const maxPageSize = 100

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if p := c.Query("page_size"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
