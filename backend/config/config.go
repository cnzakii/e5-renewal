package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host       string `yaml:"host" json:"host"`
	Port       int    `yaml:"port" json:"port"`
	PathPrefix string `yaml:"path_prefix" json:"path_prefix"`
	TLSCert    string `yaml:"tls_cert" json:"tls_cert"`
	TLSKey     string `yaml:"tls_key" json:"tls_key"`
}

type DatabaseConfig struct {
	Path string `yaml:"path" json:"path"`
}

type SecurityConfig struct {
	JWTSecret     string `yaml:"jwt_secret" json:"jwt_secret"`
	LoginKey      string `yaml:"login_key" json:"login_key"`
	EncryptionKey string `yaml:"encryption_key" json:"encryption_key"`
}

type SchedulerConfig struct {
	MinHours        int  `yaml:"min_hours" json:"min_hours"`
	MaxHours        int  `yaml:"max_hours" json:"max_hours"`
	EndpointsMin    int  `yaml:"endpoints_min" json:"endpoints_min"`
	EndpointsMax    int  `yaml:"endpoints_max" json:"endpoints_max"`
	RealisticTiming bool `yaml:"realistic_timing" json:"realistic_timing"`
}

type Config struct {
	Server    ServerConfig    `yaml:"server" json:"server"`
	Database  DatabaseConfig  `yaml:"database" json:"database"`
	Security  SecurityConfig  `yaml:"security" json:"security"`
	Scheduler SchedulerConfig `yaml:"scheduler" json:"scheduler"`
}

var (
	globalCfg    *Config
	defaultFiles = []string{"config.yaml", "config.yml", "config.json"}
)

// MustInit loads configuration and stores the singleton. Panics on error.
func MustInit(path ...string) {
	cfg, err := LoadConfig(path...)
	if err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}
	globalCfg = cfg
}

// Get returns the singleton Config instance.
func Get() *Config {
	return globalCfg
}

// LoadConfig loads configuration with the following priority:
//  1. Resolve config file path: explicit path > E5_CONFIG env > auto-detect in current dir
//  2. Parse the file (YAML or JSON based on extension)
//  3. Overlay environment variables (env vars always win)
//  4. Apply defaults, then validate
func LoadConfig(path ...string) (*Config, error) {
	filePath := resolveConfigPath(path...)

	var cfg Config
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", filePath, err)
		}
		if err := unmarshal(filePath, data, &cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", filePath, err)
		}
	}

	applyEnv(&cfg)
	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Security.JWTSecret == "" {
		return fmt.Errorf("jwt_secret must be set")
	}
	if len(cfg.Security.JWTSecret) < 16 {
		return fmt.Errorf("jwt_secret must be at least 16 characters")
	}
	if cfg.Security.EncryptionKey == "" {
		return fmt.Errorf("encryption_key must be set")
	}
	if len(cfg.Security.EncryptionKey) < 16 {
		return fmt.Errorf("encryption_key must be at least 16 characters")
	}
	return nil
}

// resolveConfigPath determines which config file to use.
func resolveConfigPath(explicit ...string) string {
	// 1. Explicit path passed by caller
	if len(explicit) > 0 && explicit[0] != "" {
		return explicit[0]
	}
	// 2. E5_CONFIG environment variable
	if v := os.Getenv("E5_CONFIG"); v != "" {
		return v
	}
	// 3. Auto-detect: config.yaml / config.yml / config.json in current dir
	for _, name := range defaultFiles {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	return ""
}

// unmarshal parses data as YAML or JSON based on file extension.
func unmarshal(path string, data []byte, cfg *Config) error {
	switch filepath.Ext(path) {
	case ".json":
		return json.Unmarshal(data, cfg)
	default: // .yaml, .yml, or anything else
		return yaml.Unmarshal(data, cfg)
	}
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("E5_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("E5_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("E5_PATH_PREFIX"); v != "" {
		cfg.Server.PathPrefix = v
	}
	if v := os.Getenv("E5_TLS_CERT"); v != "" {
		cfg.Server.TLSCert = v
	}
	if v := os.Getenv("E5_TLS_KEY"); v != "" {
		cfg.Server.TLSKey = v
	}
	if v := os.Getenv("E5_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("E5_JWT_SECRET"); v != "" {
		cfg.Security.JWTSecret = v
	}
	if v := os.Getenv("E5_LOGIN_KEY"); v != "" {
		cfg.Security.LoginKey = v
	}
	if v := os.Getenv("E5_ENCRYPTION_KEY"); v != "" {
		cfg.Security.EncryptionKey = v
	}
	// Legacy: support PORT env var (e.g. from container platforms)
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "data/e5.db"
	}
	if cfg.Scheduler.MinHours == 0 {
		cfg.Scheduler.MinHours = 2
	}
	if cfg.Scheduler.MaxHours == 0 {
		cfg.Scheduler.MaxHours = 6
	}
	if cfg.Scheduler.EndpointsMin == 0 {
		cfg.Scheduler.EndpointsMin = 3
	}
	if cfg.Scheduler.EndpointsMax == 0 {
		cfg.Scheduler.EndpointsMax = 8
	}
}

// Addr returns the "host:port" listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
