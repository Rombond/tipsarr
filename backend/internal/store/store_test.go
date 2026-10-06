package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateIdempotentAndSettings(t *testing.T) {
	ctx := context.Background()
	path := "sqlite:" + filepath.Join(t.TempDir(), "x.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "k", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "k", "v2"); err != nil { // update path
		t.Fatal(err)
	}
	if v, _ := s.GetSetting(ctx, "k"); v != "v2" {
		t.Fatalf("got %q", v)
	}
	if v, _ := s.GetSetting(ctx, "missing"); v != "" {
		t.Fatalf("got %q", v)
	}
	_ = s.Close()

	s, err = Open(ctx, path) // re-open: migrations must not re-run
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v, _ := s.GetSetting(ctx, "k"); v != "v2" {
		t.Fatalf("after reopen got %q", v)
	}
}

func TestSplitStatements(t *testing.T) {
	got := splitStatements("-- c\nCREATE TABLE a (x INT);\n\nCREATE INDEX i ON a (x);\n")
	if len(got) != 2 {
		t.Fatalf("got %d: %v", len(got), got)
	}
}

// External engines are exercised only when a URL is provided, e.g.
//
//	TIPSARR_TEST_PG_URL=postgres://t:t@localhost:55432/t TIPSARR_TEST_MYSQL_URL=mysql://t:t@localhost:53306/t go test ./internal/store
func TestMigrateExternalEngines(t *testing.T) {
	for _, env := range []string{"TIPSARR_TEST_PG_URL", "TIPSARR_TEST_MYSQL_URL"} {
		url := os.Getenv(env)
		if url == "" {
			continue
		}
		t.Run(env, func(t *testing.T) {
			ctx := context.Background()
			s, err := Open(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			u, err := s.UpsertUser(ctx, "u1", "alice", true)
			if err != nil || u.Role != RoleAdmin {
				t.Fatalf("upsert: %v %+v", err, u)
			}
			if err := s.SetSetting(ctx, "k", "v"); err != nil {
				t.Fatal(err)
			}
			if err := s.SetSetting(ctx, "k", "v2"); err != nil {
				t.Fatal(err)
			}
			if v, _ := s.GetSetting(ctx, "k"); v != "v2" {
				t.Fatalf("got %q", v)
			}
			if err := s.CreateSession(ctx, &Session{ID: "s1", UserID: "u1", CreatedAt: 1, ExpiresAt: 9999999999}); err != nil {
				t.Fatal(err)
			}
			if got, err := s.SessionUser(ctx, "s1"); err != nil || got.ID != "u1" {
				t.Fatalf("session user: %v %+v", err, got)
			}
			s2, err := Open(ctx, url) // idempotent re-open
			if err != nil {
				t.Fatal(err)
			}
			s2.Close()
		})
	}
}
