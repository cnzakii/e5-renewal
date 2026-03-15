package spa_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"e5-renewal/backend/spa"
)

func newTestFS() fstest.MapFS {
	return fstest.MapFS{
		"static/dist/index.html": &fstest.MapFile{
			Data: []byte("<html><head></head><body><h1>Test SPA</h1></body></html>"),
		},
		"static/dist/assets/app.js": &fstest.MapFile{
			Data: []byte(`console.log("test app");`),
		},
	}
}

func setupSPAEngine(prefix, pathPrefix string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	spa.RegisterSPA(r, prefix, newTestFS(), pathPrefix)
	return r
}

func TestSPA_ServeStaticFile(t *testing.T) {
	r := setupSPAEngine("", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/assets/app.js", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "console.log")
}

func TestSPA_FallbackToIndexHTML(t *testing.T) {
	r := setupSPAEngine("", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/some/unknown/route", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<html>")
	assert.Contains(t, w.Body.String(), "</head>")
}

func TestSPA_PathPrefixInjection(t *testing.T) {
	r := setupSPAEngine("", "/myapp")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `window.__E5_CONFIG__=`)
	assert.Contains(t, body, `"pathPrefix":"/myapp"`)
}

func TestSPA_PathPrefixEmpty(t *testing.T) {
	r := setupSPAEngine("", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `window.__E5_CONFIG__=`)
	assert.Contains(t, body, `"pathPrefix":""`)
}

func TestSPA_WithPrefix_MatchingRoute(t *testing.T) {
	r := setupSPAEngine("/app", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/app/dashboard", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<html>")
}

func TestSPA_WithPrefix_StaticAsset(t *testing.T) {
	r := setupSPAEngine("/app", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/app/assets/app.js", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "console.log")
}

func TestSPA_WithPrefix_OutsidePrefix(t *testing.T) {
	r := setupSPAEngine("/app", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/other/path", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSPA_RootPath(t *testing.T) {
	r := setupSPAEngine("", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<html>")
}

func TestSPA_WithPrefix_RedirectToTrailingSlash(t *testing.T) {
	r := setupSPAEngine("/app", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/app", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMovedPermanently, w.Code)
	assert.Equal(t, "/app/", w.Header().Get("Location"))
}

func TestSPA_WithPrefix_TrailingSlash(t *testing.T) {
	r := setupSPAEngine("/app", "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/app/", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<html>")
}

func TestSPA_SpecialCharsInPathPrefix(t *testing.T) {
	r := setupSPAEngine("", "/path/<with>&special")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// json.Marshal safely escapes special characters
	assert.Contains(t, body, `window.__E5_CONFIG__=`)
	assert.Contains(t, body, `pathPrefix`)
}
