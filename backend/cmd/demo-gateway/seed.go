package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/demo"
)

// The template is rebuilt when this changes (or the internal address, which is saved in it).
const seedVersion = "1"

type seedClient struct {
	base string
	c    *http.Client
}

func newSeedClient(base string) *seedClient {
	jar, _ := cookiejar.New(nil)
	return &seedClient{base: base, c: &http.Client{Jar: jar, Timeout: 90 * time.Second}}
}

func (s *seedClient) do(method, path string, body any) (int, string) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = strings.NewReader(string(b))
	}
	req, _ := http.NewRequest(method, s.base+"/api/v1"+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.c.Do(req)
	if err != nil {
		slog.Warn("seed call failed", "path", path, "err", err)
		return 0, ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		slog.Warn("seed call refused", "method", method, "path", path, "status", resp.StatusCode, "body", string(b[:min(len(b), 200)]))
	}
	return resp.StatusCode, string(b)
}

func field(body, key string) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	v, _ := m[key].(string)
	return v
}

type obj = map[string]any

// stamp identifies a template: rebuilt when the seed, the binary or the internal address changes.
func stamp(bin, internal string) string {
	st, _ := os.Stat(bin)
	var mod int64
	if st != nil {
		mod = st.ModTime().Unix()
	}
	return fmt.Sprintf("seed=%s internal=%s bin=%d", seedVersion, internal, mod)
}

// ensureTemplate builds the template data directory unless an up-to-date one exists.
func ensureTemplate(p *procSpawner, dir string) error {
	want := stamp(p.Bin, p.Internal)
	if b, err := os.ReadFile(filepath.Join(dir, ".stamp")); err == nil && string(b) == want {
		slog.Info("demo template up to date", "dir", dir)
		return nil
	}
	slog.Info("building the demo template (a minute or two)", "dir", dir)
	work := dir + ".building"
	os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o700); err != nil {
		return err
	}
	c, err := p.start(work)
	if err != nil {
		return err
	}
	seed(c.Addr, p.Internal)
	c.kill() // gracefully: SQLite checkpoints its journal into the database
	keep := work
	os.RemoveAll(filepath.Join(keep, "trailers"))
	if err := os.WriteFile(filepath.Join(keep, ".stamp"), []byte(want), 0o600); err != nil {
		return err
	}
	os.RemoveAll(dir)
	return os.Rename(keep, dir)
}

// seed fills a fresh sandbox through its API: accounts and rights, Radarr/Sonarr, the library and history,
// requests in several states, an issue with comments, watchlists and two (disabled) webhooks.
func seed(addr, internal string) {
	base := "http://" + addr
	admin := newSeedClient(base)
	if code, body := admin.do("POST", "/auth/demo", obj{"role": "admin"}); code != 200 {
		slog.Error("seed: admin sign-in failed", "status", code, "body", body)
		return
	}
	users := map[string]*seedClient{"alice": admin}
	for _, n := range []string{"bob", "carol", "dave"} {
		u := newSeedClient(base)
		u.do("POST", "/auth/login", obj{"username": n, "password": demo.Password})
		users[n] = u
	}
	bob, carol, dave := users["bob"], users["carol"], users["dave"]

	admin.do("PUT", "/admin/settings", obj{"boxofficeRegions": "US,FR", "defaultLanguage": ""})
	in := "http://" + internal
	admin.do("POST", "/admin/servarr", obj{"kind": "radarr", "name": "Radarr", "url": in + demo.PathRadarr, "apiKey": demo.ArrKey, "qualityProfileId": 4, "rootFolder": "/data/media/movies", "isDefault": true})
	admin.do("POST", "/admin/servarr", obj{"kind": "sonarr", "name": "Sonarr", "url": in + demo.PathSonarr, "apiKey": demo.ArrKey, "qualityProfileId": 4, "rootFolder": "/data/media/tv", "isDefault": true})

	ids := map[string]string{}
	var list []struct{ ID, Name string }
	_, body := admin.do("GET", "/admin/users", nil)
	_ = json.Unmarshal([]byte(body), &list)
	for _, u := range list {
		ids[u.Name] = u.ID
	}
	const (
		manageRequests = 16
		request        = 32
		requestMovie   = 262144
		requestTV      = 524288
		request4K      = 1024
		createIssues   = 4194304
		defaults       = request | requestMovie | requestTV | request4K | createIssues
	)
	access := func(name string, perms int, quota obj) {
		admin.do("PATCH", "/admin/users/"+ids[name], obj{"permissions": perms, "quota": quota})
	}
	access("bob", defaults, obj{"movieLimit": 5, "movieDays": 7})
	access("carol", defaults|manageRequests, obj{})
	access("dave", requestMovie|createIssues, obj{"movieLimit": 1, "movieDays": 1})

	for _, job := range []string{"library-sync", "history-sync", "playback-sync"} {
		admin.do("POST", "/admin/sync/"+job, nil)
		waitJob(admin, job, 3*time.Minute)
	}

	ask := func(u *seedClient, typ string, id int, seasons ...int) string {
		b := obj{"type": typ, "tmdbId": id}
		if len(seasons) > 0 {
			b["seasons"] = seasons
		}
		_, out := u.do("POST", "/requests", b)
		return field(out, "id")
	}
	ask(bob, "movie", 693134)                      // Dune: Part Two, pending
	ask(bob, "tv", 76479, 1, 2)                    // The Boys, pending
	if id := ask(bob, "movie", 335984); id != "" { // Blade Runner 2049, approved (dry-run)
		admin.do("POST", "/requests/"+id+"/approve", obj{})
	}
	if id := ask(bob, "movie", 299534); id != "" { // Avengers: Endgame, declined
		carol.do("POST", "/requests/"+id+"/decline", obj{"reason": "Nous avons déjà la version 4K."})
	}
	ask(carol, "movie", 324857) // a request manager: approved at once
	ask(dave, "movie", 545611)
	ask(admin, "tv", 1668, 1)

	_, out := bob.do("POST", "/issues", obj{"type": "movie", "tmdbId": 27205, "kind": "subtitles", "message": "Les sous-titres français sont décalés après 40 minutes."})
	if id := field(out, "id"); id != "" {
		admin.do("POST", "/issues/"+id+"/comments", obj{"message": "Merci, je regarde le fichier de sous-titres."})
		bob.do("POST", "/issues/"+id+"/comments", obj{"message": "Ça commence vers 41:30."})
	}
	_, out = carol.do("POST", "/issues", obj{"type": "tv", "tmdbId": 1396, "kind": "video", "season": 1, "episode": 3, "message": "L'image se fige au début de l'épisode."})
	if id := field(out, "id"); id != "" {
		admin.do("POST", "/issues/"+id+"/comments", obj{"message": "Fichier remplacé, ça devrait aller."})
		admin.do("POST", "/issues/"+id+"/resolve", obj{})
	}

	for _, w := range []struct {
		u    *seedClient
		typ  string
		id   int
		list string
	}{
		{bob, "movie", 274, "watchlist"}, {bob, "movie", 807, "watchlist"}, {bob, "tv", 87108, "watchlist"}, {bob, "movie", 597, "blocklist"},
		{admin, "movie", 496243, "watchlist"}, {carol, "tv", 100088, "watchlist"},
	} {
		w.u.do("POST", "/"+w.list, obj{"type": w.typ, "tmdbId": w.id})
	}

	admin.do("POST", "/admin/webhooks", obj{"name": "Discord (exemple)", "url": "https://discord.com/api/webhooks/000/demo", "enabled": false, "preset": "discord",
		"events": []string{"request.created", "request.approved", "media.available"}})
	admin.do("POST", "/admin/webhooks", obj{"name": "ntfy (exemple)", "url": "https://ntfy.sh/tipsarr-demo", "enabled": false, "preset": "ntfy",
		"events": []string{"issue.created", "issue.resolved"}})
	slog.Info("demo template seeded")
}

func waitJob(s *seedClient, name string, max time.Duration) {
	deadline := time.Now().Add(max)
	time.Sleep(time.Second)
	for time.Now().Before(deadline) {
		_, body := s.do("GET", "/admin/sync", nil)
		var st struct {
			Jobs []struct {
				Name, Status string
				Running      bool
			}
		}
		_ = json.Unmarshal([]byte(body), &st)
		for _, j := range st.Jobs {
			if j.Name == name && !j.Running && j.Status != "running" && j.Status != "never" {
				return
			}
		}
		time.Sleep(time.Second)
	}
	slog.Warn("seed: job still running", "job", name)
}
