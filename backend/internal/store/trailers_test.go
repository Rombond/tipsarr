package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Trailer file queries on every engine (set TIPSARR_TEST_PG_URL / TIPSARR_TEST_MYSQL_URL to include them).
func TestTrailerQueriesAllEngines(t *testing.T) {
	urls := map[string]string{"sqlite": "sqlite:" + filepath.Join(t.TempDir(), "tf.db")}
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
			tag := fmt.Sprintf("%d", time.Now().UnixNano()%1e9) // external DBs persist between runs
			a, b := "a"+tag, "b"+tag
			now := time.Now().Unix()

			if created, err := s.AddTrailerFile(ctx, a, now); err != nil || !created {
				t.Fatalf("add = %v %v", created, err)
			}
			if created, err := s.AddTrailerFile(ctx, a, now); err != nil || created {
				t.Fatalf("second add = %v %v", created, err)
			}
			_, _ = s.AddTrailerFile(ctx, b, now)
			f, _ := s.TrailerFileByKey(ctx, a)
			if f == nil || f.Status != TrailerQueued || f.RequestedAt != now {
				t.Fatalf("row = %+v", f)
			}
			if f, _ := s.TrailerFileByKey(ctx, "missing"+tag); f != nil {
				t.Fatalf("missing key = %+v", f)
			}

			f.Status, f.Bytes, f.Height, f.Duration, f.ReadyAt = TrailerReady, 1234, 720, 31, now
			if err := s.SaveTrailerFile(ctx, f); err != nil {
				t.Fatal(err)
			}
			g, _ := s.TrailerFileByKey(ctx, b)
			g.Status, g.Error, g.Attempts, g.NextTryAt = TrailerFailed, "HTTP 403", 1, now-1
			_ = s.SaveTrailerFile(ctx, g)

			if rows, err := s.TrailerFilesByStatus(ctx, 10, TrailerQueued, TrailerRunning); err != nil {
				t.Fatal(err)
			} else {
				for _, r := range rows {
					if r.YoutubeKey == a || r.YoutubeKey == b {
						t.Fatalf("queued list holds %+v", r)
					}
				}
			}
			retry, err := s.TrailerFilesToRetry(ctx, now, now-60, 100)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, r := range retry {
				found = found || r.YoutubeKey == b
			}
			if !found {
				t.Fatalf("temporary failure missing from the retry list: %+v", retry)
			}

			if err := s.TouchTrailerFile(ctx, a, now); err != nil {
				t.Fatal(err)
			}
			keys, err := s.RecentTrailerKeys(ctx, now, 10)
			if err != nil || len(keys) == 0 || keys[0] != a {
				t.Fatalf("recent = %v %v", keys, err)
			}
			counts, err := s.TrailerCounts(ctx)
			if err != nil || counts[TrailerReady] < 1 || counts[TrailerFailed] < 1 {
				t.Fatalf("counts = %v %v", counts, err)
			}
			if err := s.DeleteTrailerFiles(ctx); err != nil {
				t.Fatal(err)
			}
			if f, _ := s.TrailerFileByKey(ctx, a); f != nil {
				t.Fatalf("row survived the delete: %+v", f)
			}
		})
	}
}
