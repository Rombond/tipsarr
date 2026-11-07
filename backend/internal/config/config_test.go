package config

import (
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	for _, k := range []string{"TIPSARR_JELLYFIN_URL", "TIPSARR_SETUP_TOKEN", "TIPSARR_COOKIE_SECURE", "TIPSARR_SESSION_TTL", "TIPSARR_SESSION_RECHECK_GRACE"} {
		t.Setenv(k, "")
	}
	c := Load()
	if !c.RequireSetupToken || c.SecureCookies || c.SessionTTL != 240*time.Hour || c.RecheckGrace != 24*time.Hour || c.TrustedProxies == "" {
		t.Fatalf("defaults = %+v", c)
	}
	t.Setenv("TIPSARR_JELLYFIN_URL", "http://jf:8096")
	if Load().RequireSetupToken {
		t.Fatal("a preset Jellyfin URL turns the setup token off by default")
	}
	t.Setenv("TIPSARR_SETUP_TOKEN", "true")
	t.Setenv("TIPSARR_COOKIE_SECURE", "true")
	t.Setenv("TIPSARR_SESSION_TTL", "7d")
	t.Setenv("TIPSARR_SESSION_RECHECK_GRACE", "bogus")
	t.Setenv("TIPSARR_TRUSTED_PROXIES", "")
	c = Load()
	if !c.RequireSetupToken || !c.SecureCookies || c.SessionTTL != 7*24*time.Hour || c.RecheckGrace != 24*time.Hour || c.TrustedProxies != "" {
		t.Fatalf("overrides = %+v", c)
	}
}
