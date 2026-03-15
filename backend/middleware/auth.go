package middleware

import (
	"net/http"
	"strings"

	"e5-renewal/backend/config"
	"e5-renewal/backend/services/security"

	"github.com/gin-gonic/gin"
)

// RequireAuth validates JWT from the Authorization header using the global jwtSecret.
func RequireAuth() gin.HandlerFunc {
	secret := []byte(config.Get().Security.JWTSecret)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := security.ParseJWT(secret, token); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Next()
	}
}
