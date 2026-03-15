package handlers

import (
	"net/http"

	"e5-renewal/backend/config"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/services/login"
	"e5-renewal/backend/services/security"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Key string `json:"key"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func RegisterAuthRoutes(r *gin.Engine) {
	prefix := config.Get().Server.PathPrefix
	r.POST(prefix+"/api/login", middleware.LoginRateLimit(), loginHandler())
}

func loginHandler() gin.HandlerFunc {
	jwtSecret := []byte(config.Get().Security.JWTSecret)
	loginKey := login.Key()
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		if req.Key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		}

		if !security.VerifyPassword(loginKey, req.Key) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := security.SignJWT(jwtSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
			return
		}

		c.JSON(http.StatusOK, loginResponse{Token: token})
	}
}
