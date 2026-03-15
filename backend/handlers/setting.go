package handlers

import (
	"encoding/json"
	"net/http"

	"e5-renewal/backend/config"
	"e5-renewal/backend/database"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/models"
	"e5-renewal/backend/services/notifier"

	"github.com/gin-gonic/gin"
)

func RegisterSettingRoutes(r *gin.Engine) {
	prefix := config.Get().Server.PathPrefix
	group := r.Group(prefix + "/api")
	group.Use(middleware.RequireAuth())
	group.PUT("/settings/notification", updateNotificationSettingsHandler())
	group.GET("/settings/notification", getNotificationSettingsHandler())
	group.POST("/settings/notification/test", testNotificationSettingsHandler())
}

func updateNotificationSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.NotificationConfig
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		valueBytes, err := json.Marshal(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode setting value"})
			return
		}

		if err := database.Settings.Upsert(c.Request.Context(), models.SettingKeyNotification, string(valueBytes)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save setting"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "updated"})
	}
}

func getNotificationSettingsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := database.Settings.Get(c.Request.Context(), models.SettingKeyNotification)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query setting"})
			return
		}
		if raw == "" {
			c.JSON(http.StatusOK, models.NotificationConfig{
				Language:         "zh",
				ExpiryDaysBefore: 7,
				HealthThreshold:  50,
			})
			return
		}

		var value models.NotificationConfig
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse setting value"})
			return
		}

		c.JSON(http.StatusOK, value)
	}
}

func testNotificationSettingsHandler() gin.HandlerFunc {
	svc := notifier.NewService()
	return func(c *gin.Context) {
		raw, err := database.Settings.Get(c.Request.Context(), models.SettingKeyNotification)
		if err != nil || raw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "notification setting not found"})
			return
		}

		var value models.NotificationConfig
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse setting value"})
			return
		}
		if value.URL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "notification url is empty"})
			return
		}
		lang := value.Language
		if lang == "" {
			lang = "zh"
		}
		title, msg := notifier.FormatTest(lang)
		if err := svc.Send(value.URL, title, msg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "sent"})
	}
}
