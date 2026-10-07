// Package ratings shows the external scores of a title. IMDb, Metacritic and Rotten Tomatoes come
// from the default Radarr instance (a read-only lookup: Radarr already asks its own metadata
// service, so no key and no extra source). When Radarr has no Rotten Tomatoes score, or the title is
// a show (Radarr only knows movies), Rotten Tomatoes' own search is asked the way Seerr does it
// (see rottentomatoes.go). TMDB's own score is already part of the title's details.
package ratings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	hitTTL  = 24 * time.Hour
	missTTL = 15 * time.Minute // Radarr down or no score yet: ask again soon
	rtMiss  = time.Hour
)

type Score struct {
	Value float64 `json:"value" doc:"IMDb: out of 10. Metacritic and Rotten Tomatoes: out of 100"`
	Votes int     `json:"votes,omitempty"`
}

type MovieScores struct {
	IMDb                   *Score `json:"imdb,omitempty"`
	IMDbURL                string `json:"imdbUrl,omitempty"`
	Metacritic             *Score `json:"metacritic,omitempty"`
	RottenTomatoes         *Score `json:"rottenTomatoes,omitempty" doc:"Critics score in percent"`
	RottenTomatoesAudience *Score `json:"rottenTomatoesAudience,omitempty" doc:"Audience score in percent (only from Rotten Tomatoes' own search)"`
	RottenTomatoesURL      string `json:"rottenTomatoesUrl,omitempty"`
}

type radarrEntry struct {
	at      time.Time
	ttl     time.Duration
	ratings *servarr.MovieRatings // nil: Radarr could not tell
}

type rtEntry struct {
	at    time.Time
	ttl   time.Duration
	score *RTScore
}

type Service struct {
	store      *store.Store
	newServarr func(*store.ServarrInstance) *servarr.Client
	http       *http.Client
	rtToken    string
	// rtLookup is swapped in tests
	rtLookup func(ctx context.Context, kind, name string, year int) (*RTScore, error)

	mu     sync.Mutex
	radarr map[int]radarrEntry
	rt     map[string]rtEntry
}

func New(s *store.Store) *Service {
	tok := make([]byte, 16)
	_, _ = rand.Read(tok)
	svc := &Service{
		store: s, radarr: map[int]radarrEntry{}, rt: map[string]rtEntry{},
		http: &http.Client{Timeout: 10 * time.Second}, rtToken: hex.EncodeToString(tok),
		// reading only: a dry-run client is the safe choice and changes nothing for reads
		newServarr: func(in *store.ServarrInstance) *servarr.Client { return servarr.New(in.Kind, in.URL, in.APIKey, true) },
	}
	svc.rtLookup = func(ctx context.Context, kind, name string, year int) (*RTScore, error) {
		return rottenTomatoes(ctx, svc.http, svc.rtToken, kind, name, year)
	}
	return svc
}

// Movie returns the scores of a movie. title and year (TMDB's, optional) let Rotten Tomatoes be
// searched when Radarr has no score for it; withRT=false skips that search (posters that do not
// show Rotten Tomatoes do not need it). Nothing found is an empty result, never an error.
func (s *Service) Movie(ctx context.Context, tmdbID int, title string, year int, withRT bool) MovieScores {
	r := s.radarrRatings(ctx, tmdbID)
	var out MovieScores
	if r != nil {
		if r.IMDb != nil {
			out.IMDb = &Score{Value: r.IMDb.Value, Votes: r.IMDb.Votes}
			if r.IMDbID != "" {
				out.IMDbURL = "https://www.imdb.com/title/" + r.IMDbID
			}
		}
		if r.Metacritic != nil {
			out.Metacritic = &Score{Value: r.Metacritic.Value}
		}
		if r.RottenTomatoes != nil {
			out.RottenTomatoes = &Score{Value: r.RottenTomatoes.Value}
		}
		if title == "" {
			title, year = r.Title, r.Year
		}
	}
	if out.RottenTomatoes == nil && withRT && title != "" {
		if rt := s.rottenTomatoes(ctx, "movie", tmdbID, title, year); rt != nil {
			out.RottenTomatoes, out.RottenTomatoesAudience, out.RottenTomatoesURL = rt.scores()
		}
	}
	return out
}

// Show returns the Rotten Tomatoes scores of a show (Radarr has nothing for shows).
func (s *Service) Show(ctx context.Context, tmdbID int, name string, year int) MovieScores {
	var out MovieScores
	if name == "" {
		return out
	}
	if rt := s.rottenTomatoes(ctx, "tv", tmdbID, name, year); rt != nil {
		out.RottenTomatoes, out.RottenTomatoesAudience, out.RottenTomatoesURL = rt.scores()
	}
	return out
}

func (r *RTScore) scores() (critics, audience *Score, url string) {
	critics = &Score{Value: float64(r.Critics)}
	if r.Audience > 0 {
		audience = &Score{Value: float64(r.Audience)}
	}
	return critics, audience, r.URL
}

func (s *Service) rottenTomatoes(ctx context.Context, kind string, tmdbID int, name string, year int) *RTScore {
	key := kind + ":" + strconv.Itoa(tmdbID)
	s.mu.Lock()
	if e, ok := s.rt[key]; ok && time.Since(e.at) < e.ttl {
		s.mu.Unlock()
		return e.score
	}
	s.mu.Unlock()
	sc, err := s.rtLookup(ctx, kind, name, year)
	ttl := hitTTL
	if err != nil {
		slog.Debug("ratings: Rotten Tomatoes lookup failed", "err", err)
		sc, ttl = nil, rtMiss
	} else if sc == nil {
		ttl = rtMiss
	}
	s.mu.Lock()
	if len(s.rt) > 5000 {
		s.rt = map[string]rtEntry{}
	}
	s.rt[key] = rtEntry{at: time.Now(), ttl: ttl, score: sc}
	s.mu.Unlock()
	return sc
}

func (s *Service) radarrRatings(ctx context.Context, tmdbID int) *servarr.MovieRatings {
	s.mu.Lock()
	if e, ok := s.radarr[tmdbID]; ok && time.Since(e.at) < e.ttl {
		s.mu.Unlock()
		return e.ratings
	}
	s.mu.Unlock()

	r, ok := s.lookup(ctx, tmdbID)
	ttl := hitTTL
	if !ok {
		ttl = missTTL
	}
	s.mu.Lock()
	if len(s.radarr) > 5000 {
		s.radarr = map[int]radarrEntry{}
	}
	s.radarr[tmdbID] = radarrEntry{at: time.Now(), ttl: ttl, ratings: r}
	s.mu.Unlock()
	return r
}

const (
	batchLimit    = 40
	batchWait     = 8 * time.Second // an answer is never held longer than this; slower lookups finish in the background
	batchParallel = 4
	lookupBudget  = 30 * time.Second
)

// Movies returns the scores of several movies at once (for the posters of a page). Scores that
// are not remembered yet are looked up a few at a time; whatever is not ready within a few seconds
// is left out of the answer, and keeps being fetched so the next request finds it.
func (s *Service) Movies(ctx context.Context, ids []int, withRT bool) map[int]MovieScores {
	out := map[int]MovieScores{}
	if len(ids) > batchLimit {
		ids = ids[:batchLimit]
	}
	type result struct {
		id     int
		scores MovieScores
	}
	results := make(chan result, len(ids))
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), lookupBudget)
	go func() {
		defer cancel()
		sem := make(chan struct{}, batchParallel)
		var wg sync.WaitGroup
		for _, id := range ids {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				results <- result{id, s.Movie(bg, id, "", 0, withRT)}
			}()
		}
		wg.Wait()
	}()
	timeout := time.After(batchWait)
	for range ids {
		select {
		case r := <-results:
			out[r.id] = r.scores
		case <-timeout:
			return out
		case <-ctx.Done():
			return out
		}
	}
	return out
}

// Forget drops what is remembered about a movie (tests).
func (s *Service) Forget(tmdbID int) {
	s.mu.Lock()
	delete(s.radarr, tmdbID)
	delete(s.rt, "movie:"+strconv.Itoa(tmdbID))
	s.mu.Unlock()
}

// SetRTLookup replaces the Rotten Tomatoes search (tests).
func (s *Service) SetRTLookup(f func(ctx context.Context, kind, name string, year int) (*RTScore, error)) {
	s.rtLookup = f
}

func (s *Service) lookup(ctx context.Context, tmdbID int) (*servarr.MovieRatings, bool) {
	inst, err := s.store.DefaultServarr(ctx, "radarr")
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			slog.Debug("ratings: no Radarr instance", "err", err)
		}
		return nil, false
	}
	r, err := s.newServarr(inst).RatingsByTMDB(ctx, tmdbID)
	if err != nil {
		slog.Debug("ratings: Radarr lookup failed", "err", err)
		return nil, false
	}
	return r, r.IMDb != nil || r.Metacritic != nil || r.RottenTomatoes != nil
}
