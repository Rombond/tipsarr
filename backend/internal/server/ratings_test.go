package server_test

import (
	"net/http"
	"strings"
	"testing"
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
