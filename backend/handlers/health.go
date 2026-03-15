package handlers

import (
	"net/http"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/middleware"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

func RegisterHealthRoutes(r *gin.Engine) {
	prefix := config.Get().Server.PathPrefix
	r.GET(prefix+"/health", healthHandler())

	group := r.Group(prefix + "/health")
	group.Use(middleware.RequireAuth())
	group.GET("/detail", healthDetailHandler())
}

func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := database.GetDB(c.Request.Context()).DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unreachable"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "connected"})
	}
}

func healthDetailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		dbStatus := "connected"
		sqlDB, err := database.GetDB(ctx).DB()
		if err != nil {
			dbStatus = "unreachable"
		} else if err := sqlDB.Ping(); err != nil {
			dbStatus = "unreachable"
		}

		accountsCount, _ := database.Accounts.CountAll(ctx)

		var lastRunAt *time.Time
		accounts, _ := database.Accounts.List(ctx)
		for _, acc := range accounts {
			if last, err := database.TaskLogs.LastByAccount(ctx, acc.ID); err == nil {
				if lastRunAt == nil || last.StartedAt.After(*lastRunAt) {
					lastRunAt = &last.StartedAt
				}
			}
		}

		status := "ok"
		if dbStatus != "connected" {
			status = "error"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":         status,
			"db":             dbStatus,
			"uptime_seconds": int(time.Since(startTime).Seconds()),
			"accounts_count": accountsCount,
			"last_run_at":    lastRunAt,
		})
	}
}
