// Package config loads process configuration from the environment.
// Everything user-editable at runtime lives in the DB (settings table), not here.
package config

import (
	"os"
	"path/filepath"
	"strings"
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
	// SecureCookies forces the Secure flag on the session cookie (use it when Tipsarr is only
	// reached over HTTPS). Default: Secure only behind a proxy that sends X-Forwarded-Proto: https.
	SecureCookies bool
	// RequireSetupToken makes the first-run setup page ask for a one-time token printed in the log.
	// Off by default (private networks); turn it on if the port is reachable by strangers on a fresh install.
	RequireSetupToken bool
}

func Load() Config {
	dir := env("TIPSARR_CONFIG_DIR", "./config")
	c := Config{
		Addr:              env("TIPSARR_ADDR", ":"+env("TIPSARR_PORT", "8080")),
		ConfigDir:         dir,
		DBURL:             os.Getenv("TIPSARR_DB_URL"),
		LogLevel:          strings.ToLower(env("TIPSARR_LOG_LEVEL", "info")),
		JellyfinURL:       strings.TrimRight(os.Getenv("TIPSARR_JELLYFIN_URL"), "/"),
		DryRun:            env("TIPSARR_DRY_RUN", "true") != "false" && env("TIPSARR_DRY_RUN", "true") != "0",
		SecureCookies:     env("TIPSARR_COOKIE_SECURE", "false") == "true",
		RequireSetupToken: env("TIPSARR_SETUP_TOKEN", "false") == "true",
	}
	if c.DBURL == "" {
		c.DBURL = "sqlite:" + filepath.Join(dir, "tipsarr.db")
	}
	return c
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
