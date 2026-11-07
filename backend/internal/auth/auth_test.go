package auth

import (
	"context"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

// While Jellyfin cannot be asked, sessions keep working for the grace period, then are refused (fail closed)
// without being deleted.
func TestRecheckFailsClosedAfterGrace(t *testing.T) {
	ctx := context.Background()
	st := openStore(t)
	_ = st.SetSetting(ctx, SettingJellyfinURL, "http://127.0.0.1:1")
	_ = st.SetSetting(ctx, SettingJellyfinAPIKey, "k")
	s := New(st)
	s.SetRecheckEvery(0)
	s.SetRecheckGrace(time.Hour)
	s.SetSessionTTL(48 * time.Hour)
	now := time.Now()
	s.now = func() time.Time { return now }
	u, err := st.UpsertUser(ctx, "u1", "ann", store.AdminGrant{})
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.startSession(ctx, u, "", Device{})
	if err != nil {
		t.Fatal(err)
	}
	if _, sess, err := s.Authenticate(ctx, token); err != nil {
		t.Fatalf("first failure is inside the grace: %v", err)
	} else if exp := time.Unix(sess.ExpiresAt, 0); exp.Before(now.Add(47 * time.Hour)) {
		t.Fatalf("session TTL not applied: %v", exp)
	}
	now = now.Add(30 * time.Minute)
	s.nextChk = map[string]time.Time{}
	if _, _, err := s.Authenticate(ctx, token); err != nil {
		t.Fatalf("still inside the grace: %v", err)
	}
	now = now.Add(time.Hour)
	s.nextChk = map[string]time.Time{}
	if _, _, err := s.Authenticate(ctx, token); err == nil {
		t.Fatal("outage longer than the grace must refuse the session")
	}
	// not deleted: it works again once Jellyfin answers (simulated by clearing the failure)
	if _, err := st.LiveSession(ctx, hash(token)); err != nil {
		t.Fatal("session must be kept")
	}
	_ = st.SetSetting(ctx, SettingJellyfinAPIKey, "") // nothing to check: allowed
	s.failingSince = map[string]time.Time{}
	s.nextChk = map[string]time.Time{}
	if _, _, err := s.Authenticate(ctx, token); err != nil {
		t.Fatalf("after recovery: %v", err)
	}
}
