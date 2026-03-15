package spa

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterSPA registers the static file server and SPA fallback.
// prefix is the path prefix (e.g. "/x3k9m"), embeddedFS is an fs.FS (typically embed.FS).
func RegisterSPA(r *gin.Engine, prefix string, embeddedFS fs.FS, pathPrefix string) {
	sub, err := fs.Sub(embeddedFS, "static/dist")
	if err != nil {
		panic("embed: static/dist not found: " + err.Error())
	}

	// Read and inject index.html
	indexHTML := buildIndexHTML(sub, pathPrefix)

	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		urlPath := c.Request.URL.Path

		// Requests not under prefix: 404
		if prefix != "" && !strings.HasPrefix(urlPath, prefix) {
			c.Status(http.StatusNotFound)
			return
		}

		// Try to serve static assets directly (/assets/xxx.js etc.)
		// Strip the prefix from the path
		filePath := strings.TrimPrefix(urlPath, prefix)
		if filePath == "" {
			filePath = "/"
		}

		f, err := sub.Open(strings.TrimPrefix(filePath, "/"))
		if err == nil {
			f.Close()
			// Rewrite path for fileServer
			c.Request.URL.Path = filePath
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Not a static file → SPA fallback: return index.html with injected config
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}

func buildIndexHTML(sub fs.FS, pathPrefix string) []byte {
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return []byte("<html><body>index.html not found</body></html>")
	}
	// Inject runtime config before </head>
	safePrefix, _ := json.Marshal(pathPrefix)
	injection := `<script>window.__E5_CONFIG__={"pathPrefix":` + string(safePrefix) + `}</script>`
	result := bytes.Replace(data, []byte("</head>"), []byte(injection+"</head>"), 1)
	return result
}
