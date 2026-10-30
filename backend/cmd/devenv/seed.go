package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

type session struct {
	name string
	base string
	c    *http.Client
}

func newSession(base, name string) *session {
	jar, _ := cookiejar.New(nil)
	return &session{name: name, base: base, c: &http.Client{Jar: jar, Timeout: 60 * time.Second}}
}

func (s *session) do(method, path, body string) (int, string) {
	req, _ := http.NewRequest(method, s.base+"/api/v1"+path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.c.Do(req)
	if err != nil {
		log.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// must makes a call whose failure is only logged: seeding twice hits 409s, which is fine.
func (s *session) must(method, path, body string) string {
	code, out := s.do(method, path, body)
	if code >= 300 {
		log.Printf("  %s %s %s -> %d %.120s", s.name, method, path, code, out)
	}
	return out
}

func pretty(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func idIn(body, key string) string {
	var m map[string]any
	_ = json.Unmarshal([]byte(body), &m)
	s, _ := m[key].(string)
	return s
}

func seed(base string) {
	base = strings.TrimRight(base, "/")
	log.Printf("seeding %s", base)
	s := newSession(base, "setup")
	s.do("POST", "/setup", pretty(obj{"jellyfinUrl": "http://127.0.0.1:18096"})) // 409 when the env var already did it
	users := map[string]*session{}
	for _, n := range []string{"alice", "bob", "carol", "dave", "erin"} {
		u := newSession(base, n)
		code, body := u.do("POST", "/auth/login", pretty(obj{"username": n, "password": "test"}))
		if code != 200 {
			log.Fatalf("login %s = %d %s", n, code, body)
		}
		users[n] = u
	}
	alice, bob, carol, dave := users["alice"], users["bob"], users["carol"], users["dave"]

	alice.must("PUT", "/admin/settings", pretty(obj{"tmdbApiKey": "devkey", "jellyfinApiKey": "jfkey", "boxofficeRegions": "US,FR", "pushEnabled": false, "teasersFilesUrl": "http://127.0.0.1:18102", "teasersFilesKey": "dev-resolver-key", "teasersFilesEnabled": true}))
	for _, in := range []obj{
		{"kind": "radarr", "name": "Radarr (fake)", "url": "http://127.0.0.1:18099", "apiKey": "devkey", "qualityProfileId": 4, "rootFolder": "/data/media/movies", "isDefault": true},
		{"kind": "sonarr", "name": "Sonarr (fake)", "url": "http://127.0.0.1:18100", "apiKey": "devkey", "qualityProfileId": 4, "rootFolder": "/data/media/tv", "isDefault": true},
	} {
		alice.must("POST", "/admin/servarr", pretty(in))
	}

	// who can do what
	ids := map[string]string{}
	var list []struct{ ID, Name string }
	_, body := alice.do("GET", "/admin/users", "")
	_ = json.Unmarshal([]byte(body), &list)
	for _, u := range list {
		ids[u.Name] = u.ID
	}
	const (
		manageRequests = 16
		request        = 32
		requestMovie   = 262144
		requestTV      = 524288
		createIssues   = 4194304
		defaults       = request | requestMovie | requestTV | createIssues
	)
	setAccess := func(name string, perms int, quota obj) {
		alice.must("PATCH", "/admin/users/"+ids[name], pretty(obj{"permissions": perms, "quota": quota}))
	}
	setAccess("bob", defaults, obj{"movieLimit": 3, "movieDays": 7})
	setAccess("carol", defaults|manageRequests, obj{})
	setAccess("dave", requestMovie|createIssues, obj{"movieLimit": 1, "movieDays": 1})

	// requests that must be approved early so they have finished downloading when the person starts testing
	ask := func(u *session, typ string, id int, seasons ...int) string {
		body := obj{"type": typ, "tmdbId": id}
		if len(seasons) > 0 {
			body["seasons"] = seasons
		}
		return idIn(u.must("POST", "/requests", pretty(body)), "id")
	}
	early := ask(bob, "movie", 41)
	alice.must("POST", "/requests/"+early+"/approve", "{}")
	ask(carol, "movie", 46) // a request manager: approved right away

	// libraries and history
	for _, job := range []string{"library-sync", "history-sync", "playback-sync", "boxoffice-refresh"} {
		alice.must("POST", "/admin/sync/"+job, "")
		waitJob(alice, job, 150*time.Second)
	}

	// requests in every state
	p1 := ask(bob, "movie", 40) // stays pending
	_ = p1
	ask(bob, "tv", 216) // pending
	tv := ask(bob, "tv", 215, 1, 2)
	alice.must("POST", "/requests/"+tv+"/approve", "{}")
	bad := ask(bob, "movie", 50)
	alice.must("POST", "/requests/"+bad+"/approve", "{}") // the fake Radarr refuses this one: failed
	gone := ask(bob, "movie", 42)
	carol.must("POST", "/requests/"+gone+"/decline", pretty(obj{"reason": "We already have a better edition."}))
	ask(carol, "movie", 44) // downloading
	ask(carol, "tv", 217)
	ask(alice, "movie", 47)

	// issues
	i1 := idIn(bob.must("POST", "/issues", pretty(obj{"type": "movie", "tmdbId": 1, "kind": "subtitles", "message": "The French subtitles are out of sync after 40 minutes."})), "id")
	bob.must("POST", "/issues", pretty(obj{"type": "tv", "tmdbId": 201, "kind": "video", "season": 1, "episode": 3, "message": "The picture freezes at the start of the episode."}))
	carol.must("POST", "/issues", pretty(obj{"type": "movie", "tmdbId": 3, "kind": "audio", "message": "No sound on the right channel."}))
	dave.must("POST", "/issues", pretty(obj{"type": "movie", "tmdbId": 2, "kind": "other", "message": "Wrong poster."}))
	alice.must("POST", "/issues/"+i1+"/comments", pretty(obj{"message": "Thanks, I will check the subtitle file."}))
	alice.must("POST", "/issues/"+i1+"/resolve", "{}")

	// watchlist and hidden titles
	for _, w := range []struct {
		u    *session
		typ  string
		id   int
		list string
	}{
		{bob, "movie", 60, "watchlist"}, {bob, "movie", 61, "watchlist"}, {bob, "tv", 220, "watchlist"}, {bob, "movie", 62, "blocklist"}, {bob, "movie", 63, "blocklist"},
		{alice, "movie", 64, "watchlist"}, {carol, "tv", 221, "watchlist"},
	} {
		w.u.must("POST", "/"+w.list, pretty(obj{"type": w.typ, "tmdbId": w.id}))
	}

	fmt.Println()
	fmt.Println("account  password  tests")
	fmt.Println("alice    test      admin: everything, settings, users and permissions")
	fmt.Println("bob      test      default rights, 3 movies per 7 days (2 used), pending / failed / declined requests, watchlist")
	fmt.Println("carol    test      request manager (not admin): approves, sees all requests, no limit")
	fmt.Println("dave     test      movies only, no shows, 1 movie per day (unused), can report issues")
	fmt.Println("erin     test      default rights and no watch history: map her to alice (Users page) to try the history sources")
}

func waitJob(s *session, name string, max time.Duration) {
	deadline := time.Now().Add(max)
	time.Sleep(time.Second)
	for time.Now().Before(deadline) {
		_, body := s.do("GET", "/admin/sync", "")
		var st struct {
			Jobs []struct {
				Name, Status string
				Running      bool
			}
		}
		_ = json.Unmarshal([]byte(body), &st)
		for _, j := range st.Jobs {
			if j.Name == name && !j.Running && j.Status != "running" && j.Status != "never" {
				log.Printf("  job %s: %s", name, j.Status)
				return
			}
		}
		time.Sleep(time.Second)
	}
	log.Printf("  job %s: still running after %s, going on", name, max)
}

var _ = bytes.NewReader
