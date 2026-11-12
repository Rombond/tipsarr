package store

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/uptrace/bun"
)

func TestDefaultPermissionsAndNewUsers(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := s.DefaultPermissions(ctx); got != perms.Default {
		t.Fatalf("default = %d", got)
	}
	u, err := s.UpsertUser(ctx, "u1", "ann", AdminGrant{})
	if err != nil || perms.Mask(u.Permissions) != perms.Default || u.EffectivePermissions != int64(perms.Default) {
		t.Fatalf("new user = %+v %v", u, err)
	}
	// defaults are cleaned (no admin bit, implied flags added) and used for the next new person
	if err := s.SetDefaultPermissions(ctx, perms.Admin|perms.ManageRequests); err != nil {
		t.Fatal(err)
	}
	if got := s.DefaultPermissions(ctx); got&perms.Admin != 0 || got&perms.AutoApprove == 0 {
		t.Fatalf("stored default = %d", got)
	}
	u2, _ := s.UpsertUser(ctx, "u2", "bob", AdminGrant{})
	if perms.Mask(u2.Permissions)&perms.ManageRequests == 0 {
		t.Fatalf("second user = %d", u2.Permissions)
	}
	// an admin gets every flag whatever is stored; reading a user computes it
	a, _ := s.UpsertUser(ctx, "a1", "root", AdminGrant{Admin: true, Source: AdminFromJellyfin})
	got, err := s.GetUser(ctx, a.ID)
	if err != nil || got.EffectivePermissions != int64(perms.All) || !got.Can(perms.ManageUsers) {
		t.Fatalf("admin = %+v %v", got, err)
	}
}

func TestQuotaUsageCountsSeasonsAndSkipsDeclined(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	add := func(id, typ, status string, at int64, seasons ...int) {
		r := &Request{ID: id, MediaType: typ, TMDBID: 1, Title: id, RequestedBy: "u", Status: status, CreatedAt: at}
		if err := s.CreateRequest(ctx, r, seasons); err != nil {
			t.Fatal(err)
		}
		// CreateRequest stamps the time itself; the test needs its own
		if _, err := s.DB.NewUpdate().Model((*Request)(nil)).Set("created_at = ?", at).Where("id = ?", id).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	add("m1", "movie", StatusPending, 100)
	add("m2", "movie", StatusApproved, 200)
	add("m3", "movie", StatusDeclined, 300)
	add("m0", "movie", StatusPending, 5) // older than the window
	add("t1", "tv", StatusPending, 150, 1, 2, 3)
	add("t2", "tv", StatusFailed, 160, 1)
	if n, oldest, err := s.QuotaUsage(ctx, "u", "movie", 50, 0); err != nil || n != 2 || oldest != 100 {
		t.Fatalf("movies = %d %d %v", n, oldest, err)
	}
	if n, _, err := s.QuotaUsage(ctx, "u", "tv", 50, 0); err != nil || n != 3 {
		t.Fatalf("shows count seasons: %d %v", n, err)
	}
	if n, _, err := s.QuotaUsage(ctx, "u", "tv", 50, 2); err != nil || n != 2 {
		t.Fatalf("a show request costs at most the cap: %d %v", n, err)
	}
	add("t3", "tv", StatusPending, 170, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17)
	if n, oldest, err := s.QuotaUsage(ctx, "u", "tv", 50, 10); err != nil || n != 13 || oldest != 150 {
		t.Fatalf("17 seasons capped at 10 plus 3: %d %d %v", n, oldest, err)
	}
	if n, _, _ := s.QuotaUsage(ctx, "someone-else", "movie", 0, 0); n != 0 {
		t.Fatalf("other person = %d", n)
	}
}

// The migration turns the old "users may choose the folder" setting into the advanced-requests permission.
func TestMigrationKeepsTheFolderChoice(t *testing.T) {
	data, err := os.ReadFile("migrations/021_permissions.sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.UpsertUser(ctx, "u1", "ann", AdminGrant{}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "requests.user_folder", "true"); err != nil {
		t.Fatal(err)
	}
	// run only the data statements of the migration again (the columns exist already)
	for _, stmt := range strings.Split(string(data), ";\n") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "ALTER") {
			continue
		}
		if i := strings.Index(stmt, "UPDATE"); i < 0 && !strings.Contains(stmt, "INSERT") && !strings.Contains(stmt, "DELETE") {
			continue
		}
		stmt = stmt[strings.IndexAny(stmt, "UID"):]
		if _, err := s.DB.ExecContext(ctx, strings.TrimSuffix(stmt, ";")); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	u, _ := s.GetUser(ctx, "u1")
	if perms.Mask(u.Permissions)&perms.RequestAdvanced == 0 {
		t.Fatalf("existing user lost the folder choice: %d", u.Permissions)
	}
	if got := s.DefaultPermissions(ctx); got&perms.RequestAdvanced == 0 {
		t.Fatalf("new people lost the folder choice: %d", got)
	}
	if v, _ := s.GetSetting(ctx, "requests.user_folder"); v != "" {
		t.Fatalf("old setting still there: %q", v)
	}
}

// 027_request_4k.sql: everyone keeps 4K, and whoever auto-approves keeps auto-approving in 4K (the stored default too).
func TestMigration4KKeepsEveryoneAsBefore(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	old := perms.Request | perms.RequestMovie | perms.RequestTV | perms.CreateIssues // 4980768, the default before 4K
	plain, _ := s.UpsertUser(ctx, "u1", "ann", AdminGrant{})
	movies, _ := s.UpsertUser(ctx, "u2", "bob", AdminGrant{})
	all, _ := s.UpsertUser(ctx, "u3", "cid", AdminGrant{})
	set := func(id string, m perms.Mask) {
		if _, err := s.DB.NewUpdate().Model((*User)(nil)).Set("permissions = ?", int64(m)).Where("id = ?", id).Exec(ctx); err != nil {
			t.Fatal(err)
		}
	}
	set(plain.ID, old)
	set(movies.ID, old|perms.AutoApproveMovie)
	set(all.ID, old|perms.AutoApprove|perms.AutoApproveMovie|perms.AutoApproveTV)
	if err := s.SetSetting(ctx, SettingDefaultPermissions, strconv.FormatInt(int64(old|perms.AutoApproveTV), 10)); err != nil {
		t.Fatal(err)
	}

	body, err := migrationsFS.ReadFile("migrations/027_request_4k.sql")
	if err != nil {
		t.Fatal(err)
	}
	err = s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, stmt := range splitStatements(string(body)) {
			if strings.HasPrefix(stmt, "UPDATE") { // the column already exists: Open applied the whole file
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return err
				}
			}
		}
		return afterMigration["027_request_4k.sql"](ctx, tx)
	})
	if err != nil {
		t.Fatal(err)
	}

	four := perms.Request4K | perms.Request4KMovie | perms.Request4KTV
	for id, want := range map[string]perms.Mask{
		plain.ID:  old | four,
		movies.ID: old | perms.AutoApproveMovie | four | perms.AutoApprove4KMovie,
		all.ID:    old | perms.AutoApprove | perms.AutoApproveMovie | perms.AutoApproveTV | four | perms.AutoApprove4K | perms.AutoApprove4KMovie | perms.AutoApprove4KTV,
	} {
		u, err := s.GetUser(ctx, id)
		if err != nil || perms.Mask(u.Permissions) != want {
			t.Errorf("%s = %d, want %d (%v)", id, u.Permissions, want, err)
		}
	}
	if got := s.DefaultPermissions(ctx); got != old|perms.AutoApproveTV|four|perms.AutoApprove4KTV {
		t.Errorf("stored default = %d", got)
	}
	if Grant4K(int64(old)) != int64(perms.Default) {
		t.Error("the old default plus 4K must be the new default")
	}
}
