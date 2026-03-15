package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// SlogLogger returns a gin middleware that logs requests via slog.
func SlogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start).Round(time.Millisecond)

		attrs := []slog.Attr{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("duration", duration),
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "request", attrs...)
	}
}

// SlogRecovery returns a gin middleware that recovers from panics and logs via slog.
func SlogRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", "error", r, "method", c.Request.Method, "path", c.Request.URL.Path)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
