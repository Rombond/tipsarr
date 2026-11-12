package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Migration 032: sent or imported show requests already reached Sonarr (tvdb), other show requests are TMDB-numbered.
func TestMigrationSeasonNumberingBackfill(t *testing.T) {
	urls := map[string]string{"sqlite": "sqlite:" + filepath.Join(t.TempDir(), "sn.db")}
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
			p := fmt.Sprintf("m%d", time.Now().UnixNano()) // external DBs persist between runs
			// the rows are active requests: remove them so other store tests sharing the database do not see them
			t.Cleanup(func() {
				_, _ = s.DB.NewDelete().Model((*Request)(nil)).Where("id LIKE ?", p+"%").Exec(ctx)
				_, _ = s.DB.NewDelete().Model((*RequestSeason)(nil)).Where("request_id LIKE ?", p+"%").Exec(ctx)
			})
			rows := map[string]*Request{
				p + "sent":     {MediaType: "tv", Status: StatusApproved, SentAt: 10},
				p + "imported": {MediaType: "tv", Status: StatusApproved, Source: "sonarr"},
				p + "pending":  {MediaType: "tv", Status: StatusPending},
				p + "dry":      {MediaType: "tv", Status: StatusApproved, DryRun: 1},
				p + "movie":    {MediaType: "movie", Status: StatusPending},
			}
			for id, r := range rows {
				r.ID, r.TMDBID, r.Title = id, 990032, id
				if err := s.CreateRequest(ctx, r, nil); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.DB.NewUpdate().Model((*Request)(nil)).Set("season_numbering = ''").Where("id LIKE ?", p+"%").Exec(ctx); err != nil {
				t.Fatal(err)
			}
			body, err := migrationsFS.ReadFile("migrations/032_request_season_numbering.sql")
			if err != nil {
				t.Fatal(err)
			}
			for _, stmt := range splitStatements(string(body)) {
				if strings.HasPrefix(stmt, "UPDATE") { // the column exists already: Open applied the whole file
					if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
						t.Fatal(err)
					}
				}
			}
			want := map[string]string{p + "sent": "tvdb", p + "imported": "tvdb", p + "pending": "tmdb", p + "dry": "tmdb", p + "movie": ""}
			for id, w := range want {
				r, err := s.GetRequest(ctx, id)
				if err != nil || r.SeasonNumbering != w {
					t.Errorf("%s = %q, want %q (%v)", id, r.SeasonNumbering, w, err)
				}
			}
			if err := s.ReplaceRequestSeasons(ctx, p+"pending", []int{3, 4}); err != nil {
				t.Fatal(err)
			}
			if err := s.ReplaceRequestSeasons(ctx, p+"pending", []int{1, 2, 5}); err != nil {
				t.Fatal(err)
			}
			got, err := s.RequestSeasons(ctx, []string{p + "pending"})
			if err != nil || fmt.Sprint(got[p+"pending"]) != "[1 2 5]" {
				t.Fatalf("replaced seasons = %v %v", got, err)
			}
		})
	}
}
