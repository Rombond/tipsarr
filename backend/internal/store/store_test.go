package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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
			uid := fmt.Sprintf("u%d", time.Now().UnixNano()) // external DBs persist between runs
			sid := "s" + uid
			s, err := Open(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			u, err := s.UpsertUser(ctx, uid, "alice", true)
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
			if err := s.CreateSession(ctx, &Session{ID: sid, UserID: uid, CreatedAt: 1, ExpiresAt: 9999999999}); err != nil {
				t.Fatal(err)
			}
			if got, err := s.SessionUser(ctx, sid); err != nil || got.ID != uid {
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

// Library tables must behave the same on every engine (bulk insert chunking, upserts, IN lists).
func TestLibraryQueriesAllEngines(t *testing.T) {
	urls := map[string]string{"sqlite": "sqlite:" + filepath.Join(t.TempDir(), "lib.db")}
	if v := os.Getenv("TIPSARR_TEST_PG_URL"); v != "" {
		urls["postgres"] = v
	}
	if v := os.Getenv("TIPSARR_TEST_MYSQL_URL"); v != "" {
		urls["mysql"] = v
	}
	for name, url := range urls {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			s, err := Open(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()

			var items []LibraryItem
			for i := 1; i <= 450; i++ { // > 2 insert chunks
				items = append(items, LibraryItem{MediaType: "movie", TMDBID: int64(i), JellyfinID: "j", Title: "t"})
			}
			items = append(items, LibraryItem{MediaType: "tv", TMDBID: 7, JellyfinID: "js", Title: "show"})
			seasons := []LibrarySeason{{TMDBID: 7, SeasonNumber: 1, EpisodeCount: 8}}
			for i := 0; i < 2; i++ { // replacing twice must not collide on the PK
				if err := s.ReplaceLibrary(ctx, items, seasons); err != nil {
					t.Fatal(err)
				}
			}
			have, err := s.LibraryIDs(ctx, "movie", []int{1, 450, 451})
			if err != nil || !have[1] || !have[450] || have[451] {
				t.Fatalf("LibraryIDs = %v %v", have, err)
			}
			if m, tv, _ := s.LibraryCounts(ctx); m != 450 || tv != 1 {
				t.Fatalf("counts = %d %d", m, tv)
			}
			if got, _ := s.LibrarySeasons(ctx, 7); got[1] != 8 {
				t.Fatalf("seasons = %v", got)
			}

			uid := fmt.Sprintf("u%d", time.Now().UnixNano()) // external DBs persist between runs
			rows := []WatchHistory{{MediaType: "movie", TMDBID: 1, LastPlayedAt: 100, PlayCount: 1}}
			if changed, err := s.ReplaceUserHistory(ctx, uid, rows); err != nil || !changed {
				t.Fatalf("first history: %v %v", changed, err)
			}
			if changed, err := s.ReplaceUserHistory(ctx, uid, rows); err != nil || changed {
				t.Fatalf("same history must be unchanged: %v %v", changed, err)
			}
			rows = append(rows, WatchHistory{MediaType: "tv", TMDBID: 7, LastPlayedAt: 200, PlayCount: 3})
			if changed, _ := s.ReplaceUserHistory(ctx, uid, rows); !changed {
				t.Fatal("new title must count as changed")
			}
			if v, _ := s.HistoryVersion(ctx, uid); v != 2 {
				t.Fatalf("version = %d", v)
			}
			h, _ := s.UserHistory(ctx, uid, 1)
			if len(h) != 1 || h[0].TMDBID != 7 { // most recent first
				t.Fatalf("history = %+v", h)
			}
			if err := s.SaveJobRun(ctx, &JobRun{Name: "j", Status: "ok"}); err != nil {
				t.Fatal(err)
			}
			if err := s.SaveJobRun(ctx, &JobRun{Name: "j", Status: "error", Message: "x"}); err != nil {
				t.Fatal(err)
			}
			if runs, _ := s.JobRuns(ctx); len(runs) != 1 || runs[0].Status != "error" {
				t.Fatalf("job runs = %+v", runs)
			}
		})
	}
}

// Request, instance and webhook queries on every engine.
func TestRequestQueriesAllEngines(t *testing.T) {
	urls := map[string]string{"sqlite": "sqlite:" + filepath.Join(t.TempDir(), "req.db")}
	if v := os.Getenv("TIPSARR_TEST_PG_URL"); v != "" {
		urls["postgres"] = v
	}
	if v := os.Getenv("TIPSARR_TEST_MYSQL_URL"); v != "" {
		urls["mysql"] = v
	}
	for name, url := range urls {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			s, err := Open(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			tag := fmt.Sprintf("%d", time.Now().UnixNano()) // external DBs persist between runs
			uid := "user" + tag

			if _, err := s.UpsertUser(ctx, uid, "carol", false); err != nil {
				t.Fatal(err)
			}
			r := &Request{ID: "r" + tag, MediaType: "tv", TMDBID: 424242, Title: "Show", RequestedBy: uid, Status: StatusPending}
			if err := s.CreateRequest(ctx, r, []int{1, 3}); err != nil {
				t.Fatal(err)
			}
			if got, _ := s.RequestSeasons(ctx, []string{r.ID}); len(got[r.ID]) != 2 || got[r.ID][1] != 3 {
				t.Fatalf("seasons = %v", got)
			}
			if act, _ := s.ActiveRequestStatuses(ctx, "tv", []int{424242, 1}); act[424242] != StatusPending || len(act) != 1 {
				t.Fatalf("active = %v", act)
			}
			r.Status, r.SentAt, r.ServarrID = StatusApproved, 100, 7
			if err := s.UpdateRequest(ctx, r); err != nil {
				t.Fatal(err)
			}
			rows, total, err := s.ListRequests(ctx, RequestFilter{RequestedBy: uid, Statuses: []string{StatusApproved}, Take: 10})
			if err != nil || total != 1 || len(rows) != 1 || rows[0].ServarrID != 7 {
				t.Fatalf("list = %+v total=%d err=%v", rows, total, err)
			}
			if c, _ := s.RequestCounts(ctx, uid); c[StatusApproved] != 1 {
				t.Fatalf("counts = %v", c)
			}
			if names, _ := s.UserNames(ctx, []string{uid}); names[uid] != "carol" {
				t.Fatalf("names = %v", names)
			}
			if err := s.DeleteRequest(ctx, r.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.GetRequest(ctx, r.ID); err != ErrNotFound {
				t.Fatalf("deleted request still there: %v", err)
			}

			a := &ServarrInstance{ID: "a" + tag, Kind: "radarr", Name: "A", URL: "http://a", APIKey: "k", QualityProfileID: 1, RootFolder: "/m"}
			b := &ServarrInstance{ID: "b" + tag, Kind: "radarr", Name: "B", URL: "http://b", APIKey: "k", QualityProfileID: 1, RootFolder: "/m", IsDefault: 1}
			if err := s.SaveServarr(ctx, a); err != nil {
				t.Fatal(err)
			}
			if err := s.SaveServarr(ctx, b); err != nil {
				t.Fatal(err)
			}
			if def, err := s.DefaultServarr(ctx, "radarr"); err != nil || def.ID != b.ID {
				t.Fatalf("default = %+v %v", def, err)
			}
			_ = s.DeleteServarr(ctx, a.ID)
			_ = s.DeleteServarr(ctx, b.ID)

			w := &Webhook{ID: "w" + tag, Name: "hook", URL: "http://x", Events: "request.created", Enabled: 1}
			if err := s.SaveWebhook(ctx, w); err != nil {
				t.Fatal(err)
			}
			w.Name = "hook2"
			if err := s.SaveWebhook(ctx, w); err != nil { // update path must keep created_at
				t.Fatal(err)
			}
			if got, err := s.GetWebhook(ctx, w.ID); err != nil || got.Name != "hook2" || got.CreatedAt == 0 {
				t.Fatalf("webhook = %+v %v", got, err)
			}
			_ = s.DeleteWebhook(ctx, w.ID)
		})
	}
}

// Suggestions and box-office tables on every engine (incl. the ALTER TABLE column).
func TestSuggestionsBoxOfficeAllEngines(t *testing.T) {
	urls := map[string]string{"sqlite": "sqlite:" + filepath.Join(t.TempDir(), "sb.db")}
	if v := os.Getenv("TIPSARR_TEST_PG_URL"); v != "" {
		urls["postgres"] = v
	}
	if v := os.Getenv("TIPSARR_TEST_MYSQL_URL"); v != "" {
		urls["mysql"] = v
	}
	for name, url := range urls {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			s, err := Open(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			tag := fmt.Sprintf("%d", time.Now().UnixNano())
			uid := "su" + tag

			rowA := SuggestionRow{ID: "ra" + tag, UserID: uid, Kind: "account", SeedTitle: "Recommended for you", Position: 0, GeneratedAt: 1, HistoryVersion: 3, Personal: 1}
			rowB := SuggestionRow{ID: "rb" + tag, UserID: uid, Kind: "because", SeedType: "movie", SeedTMDBID: 9, SeedTitle: "Dune", Position: 1, GeneratedAt: 1, HistoryVersion: 3, Personal: 1}
			items := []SuggestionItem{
				{RowID: rowA.ID, Pos: 0, MediaType: "movie", TMDBID: 1, Title: "A", VoteTenths: 71, Overview: "o"},
				{RowID: rowA.ID, Pos: 1, MediaType: "tv", TMDBID: 2, Title: "B"},
				{RowID: rowB.ID, Pos: 0, MediaType: "movie", TMDBID: 3, Title: "C"},
			}
			for i := 0; i < 2; i++ { // replacing must wipe the previous rows and items
				rowA.ID, rowB.ID = fmt.Sprintf("ra%s-%d", tag, i), fmt.Sprintf("rb%s-%d", tag, i)
				items[0].RowID, items[1].RowID, items[2].RowID = rowA.ID, rowA.ID, rowB.ID
				if err := s.ReplaceSuggestions(ctx, uid, []SuggestionRow{rowA, rowB}, items); err != nil {
					t.Fatal(err)
				}
			}
			rows, byRow, err := s.SuggestionRows(ctx, uid)
			if err != nil || len(rows) != 2 || rows[0].Kind != "account" || len(byRow[rows[0].ID]) != 2 || byRow[rows[0].ID][0].VoteTenths != 71 {
				t.Fatalf("rows = %+v items = %+v err = %v", rows, byRow, err)
			}

			if _, err := s.ReplaceUserHistory(ctx, uid, []WatchHistory{{MediaType: "movie", TMDBID: 1, LastPlayedAt: 5, PlayCount: 1}}); err != nil {
				t.Fatal(err)
			}
			if w, _ := s.WatchedSet(ctx, uid); !w[WatchKey("movie", 1)] || len(w) != 1 {
				t.Fatalf("watched = %v", w)
			}
			if seeds, err := s.GlobalHistorySeeds(ctx, 5); err != nil || len(seeds) == 0 {
				t.Fatalf("global seeds = %v %v", seeds, err)
			}

			week := BoxOfficeWeek{Region: "T" + tag[len(tag)-2:], WeekKey: "2026W40", Label: "October 2-4, 2026", FetchedAt: 1}
			entries := []BoxOfficeEntry{
				{Region: week.Region, WeekKey: week.WeekKey, Pos: 1, Title: "Verity", WeekendGross: 32031011, TotalGross: 32031011, WeeksInRelease: 1, TMDBID: 900},
				{Region: week.Region, WeekKey: week.WeekKey, Pos: 2, Title: "Other"},
			}
			for i := 0; i < 2; i++ {
				if err := s.SaveBoxOffice(ctx, week, entries); err != nil {
					t.Fatal(err)
				}
			}
			w, got, err := s.BoxOffice(ctx, week.Region, week.WeekKey)
			if err != nil || w.Label != week.Label || len(got) != 2 || got[0].WeekendGross != 32031011 {
				t.Fatalf("box office = %+v %+v %v", w, got, err)
			}
			if weeks, _ := s.BoxOfficeWeeks(ctx, week.Region); len(weeks) != 1 {
				t.Fatalf("weeks = %v", weeks)
			}
			if _, _, err := s.BoxOffice(ctx, week.Region, "2000W01"); err != ErrNotFound {
				t.Fatalf("missing week err = %v", err)
			}

			inst := &ServarrInstance{ID: "g" + tag, Kind: "radarr", Name: "G", URL: "http://g", APIKey: "k", QualityProfileID: 1, RootFolder: "/m", GenreRoots: `{"28":"/a"}`}
			if err := s.SaveServarr(ctx, inst); err != nil {
				t.Fatal(err)
			}
			if got, err := s.GetServarr(ctx, inst.ID); err != nil || got.GenreRoots != `{"28":"/a"}` {
				t.Fatalf("genre roots = %+v %v", got, err)
			}
			_ = s.DeleteServarr(ctx, inst.ID)
		})
	}
}
