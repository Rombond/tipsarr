package main

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Config comes from the environment (see README.md).
type Config struct {
	Addr         string
	APIKey       string
	CacheDir     string
	CacheLimit   int64 // bytes
	MaxHeight    int   // default when a request does not say
	HourlyCap    int
	PauseMin     time.Duration
	PauseMax     time.Duration
	Timeout      time.Duration
	Workers      int
	YtDlp        string
	FFmpeg       string
	FFprobe      string
	Deno         string
	AutoUpdate   bool
	BreakerAfter int
	BreakerFor   time.Duration
	UnusedAfter  time.Duration
}

func loadConfig() (Config, error) {
	c := Config{
		Addr:         env("RESOLVER_ADDR", ":8090"),
		APIKey:       os.Getenv("RESOLVER_API_KEY"),
		CacheDir:     env("RESOLVER_CACHE_DIR", "/cache"),
		CacheLimit:   int64(envFloat("RESOLVER_CACHE_GB", 5) * (1 << 30)),
		MaxHeight:    envInt("RESOLVER_MAX_HEIGHT", 720),
		HourlyCap:    envInt("RESOLVER_HOURLY_CAP", 60),
		PauseMin:     time.Duration(envInt("RESOLVER_PAUSE_MIN_SECONDS", 5)) * time.Second,
		PauseMax:     time.Duration(envInt("RESOLVER_PAUSE_MAX_SECONDS", 15)) * time.Second,
		Timeout:      time.Duration(envInt("RESOLVER_TIMEOUT_SECONDS", 120)) * time.Second,
		Workers:      envInt("RESOLVER_WORKERS", 1),
		YtDlp:        env("RESOLVER_YTDLP", "yt-dlp"),
		FFmpeg:       env("RESOLVER_FFMPEG", "ffmpeg"),
		FFprobe:      env("RESOLVER_FFPROBE", "ffprobe"),
		Deno:         env("RESOLVER_DENO", "deno"),
		AutoUpdate:   env("RESOLVER_AUTO_UPDATE", "true") == "true",
		BreakerAfter: envInt("RESOLVER_BREAKER_AFTER", 5),
		BreakerFor:   time.Duration(envInt("RESOLVER_BREAKER_HOURS", 6)) * time.Hour,
		UnusedAfter:  time.Duration(envInt("RESOLVER_UNUSED_DAYS", 30)) * 24 * time.Hour,
	}
	if c.APIKey == "" {
		return c, errors.New("RESOLVER_API_KEY is required")
	}
	if c.PauseMax < c.PauseMin {
		c.PauseMax = c.PauseMin
	}
	c.Workers = max(c.Workers, 1)
	return c, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func envFloat(k string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(k), 64); err == nil {
		return v
	}
	return def
}
