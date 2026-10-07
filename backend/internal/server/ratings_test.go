package server_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/ratings"
)

// The IMDb / Metacritic / Rotten Tomatoes scores come from the default Radarr; a score Radarr does
// not have is left out, and without Radarr the answer is simply empty.
func TestMovieRatingsFromRadarr(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)

	if resp, _ := call(t, e.app, "GET", "/api/v1/media/movie/5/ratings", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "GET", "/api/v1/media/movie/5/ratings", "", bob)
	if resp.StatusCode != 200 || strings.Contains(body, "imdb") {
		t.Fatalf("without Radarr = %d %s", resp.StatusCode, body)
	}

	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr"))
	e.ratings.Forget(5) // the empty answer above is remembered for a few minutes
	resp, body = call(t, e.app, "GET", "/api/v1/media/movie/5/ratings", "", bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"imdb":{"value":7.8,"votes":1200}`) ||
		!strings.Contains(body, `"imdbUrl":"https://www.imdb.com/title/tt1234567"`) || !strings.Contains(body, `"metacritic":{"value":71}`) {
		t.Fatalf("with Radarr = %d %s", resp.StatusCode, body)
	}
	if strings.Contains(body, "rottenTomatoes") {
		t.Fatalf("a missing score must be left out: %s", body)
	}
}

// Posters ask for many movies at once; the answer is keyed by TMDB id.
func TestRatingsOfSeveralMovies(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr"))
	resp, body := call(t, e.app, "GET", "/api/v1/ratings/movies?ids=5,6,x,0", "", bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"5":{`) || !strings.Contains(body, `"imdb":{"value":7.8`) || !strings.Contains(body, `"6":{`) {
		t.Fatalf("batch = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "GET", "/api/v1/ratings/movies?ids=5", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous = %d", resp.StatusCode)
	}
}

// Each person picks which score they see on posters; anything else is refused.
func TestRatingSourcePreference(t *testing.T) {
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)
	resp, body := call(t, e.app, "PATCH", "/api/v1/me", `{"ratingSource":"imdb"}`, bob)
	if resp.StatusCode != 200 || !strings.Contains(body, `"ratingSource":"imdb"`) {
		t.Fatalf("set = %d %s", resp.StatusCode, body)
	}
	if resp, body := call(t, e.app, "PATCH", "/api/v1/me", `{"ratingSource":"letterboxd"}`, bob); resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("invalid = %d %s", resp.StatusCode, body)
	}
	_, body = call(t, e.app, "GET", "/api/v1/me", "", bob)
	if !strings.Contains(body, `"ratingSource":"imdb"`) {
		t.Fatalf("not kept: %s", body)
	}
	call(t, e.app, "PATCH", "/api/v1/me", `{"ratingSource":""}`, bob)
	_, body = call(t, e.app, "GET", "/api/v1/me", "", bob)
	if !strings.Contains(body, `"ratingSource":""`) {
		t.Fatalf("not cleared: %s", body)
	}
}

// Radarr first; when it has no Rotten Tomatoes score (or for shows) Rotten Tomatoes' own search is asked.
func TestRottenTomatoesFallback(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr")) // its lookup has no Rotten Tomatoes score
	call(t, e.app, "PUT", "/api/v1/admin/settings", `{"tmdbApiKey":"k123"}`, admin)

	var calls []string
	e.ratings.SetRTLookup(func(_ context.Context, kind, name string, year int) (*ratings.RTScore, error) {
		calls = append(calls, fmt.Sprintf("%s|%s|%d", kind, name, year))
		return &ratings.RTScore{Critics: 77, Audience: 90, URL: "https://www.rottentomatoes.com/m/runner_2026"}, nil
	})

	_, body := call(t, e.app, "GET", "/api/v1/media/movie/1/ratings", "", bob)
	if !strings.Contains(body, `"rottenTomatoes":{"value":77}`) || !strings.Contains(body, `"rottenTomatoesAudience":{"value":90}`) ||
		!strings.Contains(body, `"rottenTomatoesUrl":"https://www.rottentomatoes.com/m/runner_2026"`) {
		t.Fatalf("movie fallback: %s", body)
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "movie|Movie One|") {
		t.Fatalf("search = %v", calls)
	}
	// asked once only (remembered)
	call(t, e.app, "GET", "/api/v1/media/movie/1/ratings", "", bob)
	if len(calls) != 1 {
		t.Fatalf("not remembered: %v", calls)
	}

	_, body = call(t, e.app, "GET", "/api/v1/media/tv/2/ratings", "", bob)
	if !strings.Contains(body, `"rottenTomatoes":{"value":77}`) || len(calls) != 2 || !strings.HasPrefix(calls[1], "tv|Show Two|") {
		t.Fatalf("show: %s %v", body, calls)
	}

	// posters only ask Rotten Tomatoes when that is the score they show
	e.ratings.Forget(5)
	calls = nil
	call(t, e.app, "GET", "/api/v1/ratings/movies?ids=5&source=imdb", "", bob)
	if len(calls) != 0 {
		t.Fatalf("an IMDb poster must not search Rotten Tomatoes: %v", calls)
	}
	call(t, e.app, "GET", "/api/v1/ratings/movies?ids=5&source=rottenTomatoes", "", bob)
	if len(calls) != 1 || calls[0] != "movie|Request Me|2025" {
		t.Fatalf("RT poster: %v", calls)
	}
}

// Radarr's own Rotten Tomatoes score wins over the search.
func TestRadarrRottenTomatoesWins(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	arr := newFakeArr(t, "radarr")
	arr.rt.Store(64)
	addInstance(t, e, admin, "radarr", arr)
	called := false
	e.ratings.SetRTLookup(func(context.Context, string, string, int) (*ratings.RTScore, error) {
		called = true
		return nil, nil
	})
	_, body := call(t, e.app, "GET", "/api/v1/media/movie/5/ratings", "", bob)
	if !strings.Contains(body, `"rottenTomatoes":{"value":64}`) || called {
		t.Fatalf("radarr score = %s (search called: %v)", body, called)
	}
}
