package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret     = "test-jwt-secret-1234567890"
	testEncryptionKey = "test-encryption-key-1234"
)

// clearEnv unsets all E5-related environment variables for a clean test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"E5_CONFIG", "E5_HOST", "E5_PORT", "E5_PATH_PREFIX",
		"E5_TLS_CERT", "E5_TLS_KEY", "E5_DB_PATH",
		"E5_JWT_SECRET", "E5_LOGIN_KEY", "E5_ENCRYPTION_KEY", "PORT",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestLoadConfig_FromYAMLFile(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "127.0.0.1"
  port: 9090
  path_prefix: "/api"
database:
  path: "test.db"
security:
  jwt_secret: "` + testJWTSecret + `"
  login_key: "mylogin"
  encryption_key: "` + testEncryptionKey + `"
scheduler:
  min_hours: 3
  max_hours: 10
  endpoints_min: 5
  endpoints_max: 12
  realistic_timing: true
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	cfg, err := LoadConfig(yamlPath)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "/api", cfg.Server.PathPrefix)
	assert.Equal(t, "test.db", cfg.Database.Path)
	assert.Equal(t, testJWTSecret, cfg.Security.JWTSecret)
	assert.Equal(t, "mylogin", cfg.Security.LoginKey)
	assert.Equal(t, testEncryptionKey, cfg.Security.EncryptionKey)
	assert.Equal(t, 3, cfg.Scheduler.MinHours)
	assert.Equal(t, 10, cfg.Scheduler.MaxHours)
	assert.Equal(t, 5, cfg.Scheduler.EndpointsMin)
	assert.Equal(t, 12, cfg.Scheduler.EndpointsMax)
	assert.True(t, cfg.Scheduler.RealisticTiming)
}

func TestLoadConfig_FromJSONFile(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "config.json")
	content := `{
  "server": {"host": "0.0.0.0", "port": 3000},
  "security": {"jwt_secret": "` + testJWTSecret + `", "encryption_key": "` + testEncryptionKey + `"}
}`
	require.NoError(t, os.WriteFile(jsonPath, []byte(content), 0644))

	cfg, err := LoadConfig(jsonPath)
	require.NoError(t, err)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, testJWTSecret, cfg.Security.JWTSecret)
}

func TestLoadConfig_EnvOverridesFile(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "file-host"
  port: 1111
security:
  jwt_secret: "` + testJWTSecret + `"
  encryption_key: "` + testEncryptionKey + `"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	t.Setenv("E5_HOST", "env-host")
	t.Setenv("E5_PORT", "2222")
	t.Setenv("E5_DB_PATH", "env.db")
	t.Setenv("E5_LOGIN_KEY", "env-login")

	cfg, err := LoadConfig(yamlPath)
	require.NoError(t, err)

	assert.Equal(t, "env-host", cfg.Server.Host)
	assert.Equal(t, 2222, cfg.Server.Port)
	assert.Equal(t, "env.db", cfg.Database.Path)
	assert.Equal(t, "env-login", cfg.Security.LoginKey)
}

func TestLoadConfig_EnvOnly(t *testing.T) {
	clearEnv(t)

	// Change to a temp dir so no auto-detected config file interferes.
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)
	t.Setenv("E5_PORT", "4444")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 4444, cfg.Server.Port)
	assert.Equal(t, testJWTSecret, cfg.Security.JWTSecret)
}

func TestLoadConfig_Defaults(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "data/e5.db", cfg.Database.Path)
	assert.Equal(t, 2, cfg.Scheduler.MinHours)
	assert.Equal(t, 6, cfg.Scheduler.MaxHours)
	assert.Equal(t, 3, cfg.Scheduler.EndpointsMin)
	assert.Equal(t, 8, cfg.Scheduler.EndpointsMax)
}

func TestLoadConfig_LegacyPORTEnv(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)
	t.Setenv("PORT", "5555")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 5555, cfg.Server.Port)
}

func TestLoadConfig_TLSEnv(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)
	t.Setenv("E5_TLS_CERT", "/path/cert.pem")
	t.Setenv("E5_TLS_KEY", "/path/key.pem")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "/path/cert.pem", cfg.Server.TLSCert)
	assert.Equal(t, "/path/key.pem", cfg.Server.TLSKey)
}

func TestLoadConfig_PathPrefixEnv(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)
	t.Setenv("E5_PATH_PREFIX", "/prefix")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "/prefix", cfg.Server.PathPrefix)
}

func TestLoadConfig_E5ConfigEnv(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "custom.yaml")
	content := `
server:
  port: 7777
security:
  jwt_secret: "` + testJWTSecret + `"
  encryption_key: "` + testEncryptionKey + `"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))
	t.Setenv("E5_CONFIG", yamlPath)

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 7777, cfg.Server.Port)
}

func TestValidate_MissingJWTSecret(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)

	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt_secret must be set")
}

func TestValidate_ShortJWTSecret(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", "short")
	t.Setenv("E5_ENCRYPTION_KEY", testEncryptionKey)

	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt_secret must be at least 16 characters")
}

func TestValidate_MissingEncryptionKey(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)

	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption_key must be set")
}

func TestValidate_ShortEncryptionKey(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	t.Setenv("E5_JWT_SECRET", testJWTSecret)
	t.Setenv("E5_ENCRYPTION_KEY", "short")

	_, err = LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encryption_key must be at least 16 characters")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	clearEnv(t)
	_, err := LoadConfig("/nonexistent/config.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(badPath, []byte("{{invalid"), 0644))

	_, err := LoadConfig(badPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")
}

func TestMustInit_PanicsOnError(t *testing.T) {
	clearEnv(t)

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { os.Chdir(origDir) })

	assert.Panics(t, func() {
		MustInit()
	})
}

func TestMustInit_SetsSingleton(t *testing.T) {
	clearEnv(t)

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	content := `
security:
  jwt_secret: "` + testJWTSecret + `"
  encryption_key: "` + testEncryptionKey + `"
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	// Save and restore global state.
	old := globalCfg
	t.Cleanup(func() { globalCfg = old })

	MustInit(yamlPath)
	cfg := Get()
	require.NotNil(t, cfg)
	assert.Equal(t, testJWTSecret, cfg.Security.JWTSecret)
}

func TestConfig_Addr(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"default", "", 8080, ":8080"},
		{"with host", "127.0.0.1", 9090, "127.0.0.1:9090"},
		{"ipv6", "::1", 443, "::1:443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{Host: tt.host, Port: tt.port},
			}
			assert.Equal(t, tt.want, cfg.Addr())
		})
	}
}
