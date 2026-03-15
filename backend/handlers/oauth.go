package handlers

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"e5-renewal/backend/config"
	"e5-renewal/backend/middleware"
	"e5-renewal/backend/services/graph"
	"e5-renewal/backend/services/oauth"

	"github.com/gin-gonic/gin"
)

// delegatedScope builds the scope string for auth_code token requests.
func delegatedScope() string {
	return strings.Join(graph.DelegatedScopeURLs(), " ") + " offline_access"
}

func RegisterOAuthRoutes(r *gin.Engine) {
	registerOAuthRoutes(r, oauth.NewService(nil))
}

func registerOAuthRoutes(r *gin.Engine, svc *oauth.Service) {
	prefix := config.Get().Server.PathPrefix

	// OAuth callback from Microsoft — no auth required
	r.GET(prefix+"/api/oauth/callback", func(c *gin.Context) {
		code := c.Query("code")
		state := c.Query("state")
		if code == "" || state == "" {
			c.Data(http.StatusBadRequest, "text/html", oauthResultHTML("error", "missing code or state parameter", requestOrigin(c)))
			return
		}

		stateData, ok := oauth.GlobalStateStore.Consume(state)
		if !ok {
			c.Data(http.StatusBadRequest, "text/html", oauthResultHTML("error", "state invalid or expired, please re-authorize", requestOrigin(c)))
			return
		}

		origin := originFromRedirectURI(stateData.RedirectURI, c)

		tokenResp, err := svc.ExchangeAuthCode(c.Request.Context(),
			stateData.TenantID, stateData.ClientID, stateData.ClientSecret,
			code, stateData.RedirectURI, delegatedScope())
		if err != nil {
			c.Data(http.StatusBadRequest, "text/html", oauthResultHTML("error", err.Error(), origin))
			return
		}

		var tokenJSON []byte
		tokenJSON, err = json.Marshal(map[string]string{
			"refresh_token": tokenResp.RefreshToken,
			"access_token":  tokenResp.AccessToken,
		})
		if err != nil {
			c.Data(http.StatusInternalServerError, "text/html", oauthResultHTML("error", "internal error", origin))
			return
		}
		c.Data(http.StatusOK, "text/html", oauthResultHTML("success", string(tokenJSON), origin))
	})

	// Auth-protected OAuth routes
	authGroup := r.Group(prefix + "/api/oauth")
	authGroup.Use(middleware.RequireAuth())

	// Step 1: frontend requests authorize URL (POST to avoid leaking secret in query params)
	authGroup.POST("/authorize", func(c *gin.Context) {
		var req struct {
			ClientID     string `json:"client_id" binding:"required"`
			ClientSecret string `json:"client_secret"`
			TenantID     string `json:"tenant_id" binding:"required"`
			RedirectURI  string `json:"redirect_uri" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "client_id, tenant_id, and redirect_uri required"})
			return
		}

		parsed, err := url.Parse(req.RedirectURI)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid redirect_uri"})
			return
		}

		redirectURI := req.RedirectURI
		ttl := 5 * time.Minute

		state := oauth.GlobalStateStore.NewState(oauth.OAuthState{
			ClientID:     req.ClientID,
			ClientSecret: req.ClientSecret,
			TenantID:     req.TenantID,
			RedirectURI:  redirectURI,
			TTL:          ttl,
		})

		scopes := append([]string{"offline_access"}, graph.DelegatedScopeURLs()...)
		authorizeURL := svc.BuildAuthorizeURL(req.TenantID, req.ClientID, redirectURI, state, scopes)

		c.JSON(http.StatusOK, gin.H{"authorize_url": authorizeURL})
	})

	// Exchange endpoint: accepts a callback URL and exchanges the code for tokens
	authGroup.POST("/exchange", func(c *gin.Context) {
		var req struct {
			CallbackURL string `json:"callback_url" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid callback URL"})
			return
		}

		parsed, err := url.Parse(req.CallbackURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid callback URL"})
			return
		}

		query := parsed.Query()
		code := query.Get("code")
		state := query.Get("state")
		if code == "" || state == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing code or state in callback URL"})
			return
		}

		stateData, ok := oauth.GlobalStateStore.Consume(state)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "state invalid or expired, please re-authorize"})
			return
		}

		tokenResp, err := svc.ExchangeAuthCode(c.Request.Context(),
			stateData.TenantID, stateData.ClientID, stateData.ClientSecret,
			code, stateData.RedirectURI, delegatedScope())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"refresh_token": tokenResp.RefreshToken,
			"access_token":  tokenResp.AccessToken,
		})
	})
}

// originFromRedirectURI extracts the origin (scheme://host) from the stored
// redirect_uri, which reflects the real user-facing scheme even behind a
// reverse proxy. Falls back to requestOrigin if parsing fails.
func originFromRedirectURI(redirectURI string, c *gin.Context) string {
	if parsed, err := url.Parse(redirectURI); err == nil && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host
	}
	return requestOrigin(c)
}

// requestOrigin derives the origin (scheme://host) from the incoming request.
func requestOrigin(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}

// oauthResultHTML returns an HTML page that posts the OAuth result back to the
// opener window via postMessage (using a specific origin), then closes itself.
func oauthResultHTML(status, payload, origin string) []byte {
	safeStatus, _ := json.Marshal(status)
	safePayload, _ := json.Marshal(payload)
	safeOrigin, _ := json.Marshal(origin)

	displayText := map[string]string{
		"success": "Authorization successful",
		"error":   "Authorization failed: " + html.EscapeString(payload),
	}[status]

	htmlDoc := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>OAuth</title></head>
<body>
<script>
(function() {
  var result = { status: %s, payload: %s };
  if (window.opener) {
    window.opener.postMessage({ type: 'e5-oauth-result', data: result }, %s);
  }
  setTimeout(function() { window.close(); }, 500);
})();
</script>
<p style="font-family:sans-serif;text-align:center;margin-top:40px">
  %s
</p>
</body>
</html>`, safeStatus, safePayload, safeOrigin, displayText)
	return []byte(htmlDoc)
}
