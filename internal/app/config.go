package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Config contains process-wide deployment settings loaded at startup.
type Config struct {
	Address         string
	PublicURL       string
	TrustedProxies  []netip.Prefix
	Driver          string
	DSN             string
	DataDir         string
	InitialPassword string
	LogLevel        slog.Level
	LogFormat       string
	LogFile         string
}

// LoadConfig reads environment settings and validates the public URL and storage configuration.
func LoadConfig() (Config, error) {
	get := func(key, fallback string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}

		return fallback
	}
	c := Config{
		Address:         get("E5_LISTEN", "127.0.0.1:8080"),
		PublicURL:       get("E5_PUBLIC_URL", "http://127.0.0.1:8080"),
		DataDir:         get("E5_DATA_DIR", "data"),
		Driver:          get("E5_DB_DRIVER", "sqlite"),
		DSN:             os.Getenv("E5_DB_DSN"),
		InitialPassword: os.Getenv("E5_INITIAL_PASSWORD"),
		LogFormat:       get("E5_LOG_FORMAT", "text"),
		LogFile:         os.Getenv("E5_LOG_FILE"),
	}
	if err := c.LogLevel.UnmarshalText([]byte(get("E5_LOG_LEVEL", "INFO"))); err != nil {
		return c, errors.New("E5_LOG_LEVEL must be DEBUG, INFO, WARN, or ERROR")
	}

	if c.LogFormat != "text" && c.LogFormat != "json" {
		return c, errors.New("E5_LOG_FORMAT must be text or json")
	}

	u, err := parsePublicURL(c.PublicURL)
	if err != nil {
		return c, err
	}

	c.PublicURL = u.String()
	for value := range strings.SplitSeq(os.Getenv("E5_TRUSTED_PROXIES"), ",") {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			address, addressError := netip.ParseAddr(value)
			if addressError != nil {
				return c, errors.New("E5_TRUSTED_PROXIES must contain comma-separated IP addresses or CIDRs")
			}

			prefix = netip.PrefixFrom(address, address.BitLen())
		}

		c.TrustedProxies = append(c.TrustedProxies, prefix.Masked())
	}

	switch c.Driver {
	case "sqlite":
		if c.DSN == "" {
			c.DSN = filepath.Join(c.DataDir, "e5.db")
		}
	case "postgres", "mysql":
		if c.DSN == "" {
			return c, errors.New("E5_DB_DSN is required for external databases")
		}
	default:
		return c, fmt.Errorf("unsupported E5_DB_DRIVER %q", c.Driver)
	}

	return c, nil
}

func parsePublicURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil ||
		!validWebURL(u) ||
		u.Opaque != "" ||
		u.ForceQuery ||
		u.RawQuery != "" ||
		u.Fragment != "" ||
		strings.Contains(value, "#") {
		return nil, errors.New("E5_PUBLIC_URL must be an HTTP or HTTPS URL without credentials, query, or fragment")
	}

	// Keep one path spelling across the browser, proxy, router, and session cookie.
	u.Path = strings.TrimRight(u.Path, "/")
	if u.RawPath != "" ||
		u.EscapedPath() != u.Path ||
		strings.ContainsAny(u.Path, "\\;") ||
		(u.Path != "" && path.Clean(u.Path) != u.Path) {
		return nil, errors.New(
			"E5_PUBLIC_URL must use a plain path without percent encoding, dot segments, backslashes, or semicolons",
		)
	}

	// Match the browser's origin spelling, including omitted default ports.
	if (u.Scheme == "http" && u.Port() == "80") || (u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
		if strings.Contains(u.Host, ":") {
			u.Host = "[" + u.Host + "]"
		}
	}

	u.Host = strings.ToLower(u.Host)

	return u, nil
}

func validWebURL(u *url.URL) bool {
	return u.Host != "" && u.User == nil &&
		(u.Scheme == "https" || u.Scheme == "http")
}
