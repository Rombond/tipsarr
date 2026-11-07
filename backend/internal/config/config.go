// Package config loads process configuration from the environment.
// Everything user-editable at runtime lives in the DB (settings table), not here.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr      string // listen address, e.g. ":8080"
	ConfigDir string // holds the SQLite file and generated data
	DBURL     string // sqlite:<path> | postgres://... | mysql://...
	LogLevel  string
	// JellyfinURL optionally pre-seeds the Jellyfin URL so the setup step can be skipped.
	JellyfinURL string
	// DryRun (default true): never send requests/adds to Radarr/Sonarr.
	DryRun bool
	// SecureCookies forces the Secure flag on the session cookie (TIPSARR_COOKIE_SECURE=true; use it when
	// Tipsarr is only reached over HTTPS). Otherwise Secure is set when the request came over TLS or through
	// a proxy that sends X-Forwarded-Proto: https.
	SecureCookies bool
	// RequireSetupToken makes the first-run setup page ask for a one-time token printed in the log.
	// On by default unless TIPSARR_JELLYFIN_URL is set (that closes the setup page anyway);
	// TIPSARR_SETUP_TOKEN=false turns it off.
	RequireSetupToken bool
	// TrustedProxies (TIPSARR_TRUSTED_PROXIES) lists the CIDRs whose X-Forwarded-For / X-Real-IP is believed.
	// Unset = private and loopback ranges; set but empty = nobody.
	TrustedProxies string
	// SessionTTL (TIPSARR_SESSION_TTL, default 10d) is the sliding session lifetime.
	SessionTTL time.Duration
	// RecheckGrace (TIPSARR_SESSION_RECHECK_GRACE, default 24h): how long sessions keep working while
	// Jellyfin cannot be asked whether their account still exists.
	RecheckGrace time.Duration
}

func Load() Config {
	dir := env("TIPSARR_CONFIG_DIR", "./config")
	c := Config{
		Addr:          env("TIPSARR_ADDR", ":"+env("TIPSARR_PORT", "8080")),
		ConfigDir:     dir,
		DBURL:         os.Getenv("TIPSARR_DB_URL"),
		LogLevel:      strings.ToLower(env("TIPSARR_LOG_LEVEL", "info")),
		JellyfinURL:   strings.TrimRight(os.Getenv("TIPSARR_JELLYFIN_URL"), "/"),
		DryRun:        env("TIPSARR_DRY_RUN", "true") != "false" && env("TIPSARR_DRY_RUN", "true") != "0",
		SecureCookies: isTrue(os.Getenv("TIPSARR_COOKIE_SECURE")),
		SessionTTL:    duration("TIPSARR_SESSION_TTL", 10*24*time.Hour),
		RecheckGrace:  duration("TIPSARR_SESSION_RECHECK_GRACE", 24*time.Hour),
	}
	if v, ok := os.LookupEnv("TIPSARR_TRUSTED_PROXIES"); ok {
		c.TrustedProxies = v
	} else {
		c.TrustedProxies = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128,fc00::/7"
	}
	if v := os.Getenv("TIPSARR_SETUP_TOKEN"); v != "" {
		c.RequireSetupToken = isTrue(v)
	} else {
		c.RequireSetupToken = c.JellyfinURL == ""
	}
	if c.DBURL == "" {
		c.DBURL = "sqlite:" + filepath.Join(dir, "tipsarr.db")
	}
	return c
}

func isTrue(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes"
}

// duration reads a Go duration ("36h", "90m") or a number of days ("10d"); a bad value keeps def (logged).
func duration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := ParseDuration(v)
	if err != nil || d <= 0 {
		slog.Warn("ignoring a bad duration, using the default", "var", key, "value", v, "default", def.String())
		return def
	}
	return d
}

// ParseDuration is time.ParseDuration plus a "d" (days) suffix.
func ParseDuration(v string) (time.Duration, error) {
	if n, ok := strings.CutSuffix(v, "d"); ok {
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, fmt.Errorf("bad duration %q", v)
		}
		return time.Duration(f * float64(24*time.Hour)), nil
	}
	return time.ParseDuration(v)
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
