// Package ratings shows the IMDb, Metacritic and Rotten Tomatoes scores of a movie. They come from
// the default Radarr instance (a read-only lookup): Radarr already talks to its own metadata
// service, so Tipsarr needs no key and no extra source. Shows have no scores here (Radarr only
// knows movies).
package ratings

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	hitTTL  = 24 * time.Hour
	missTTL = 15 * time.Minute // Radarr down or no score yet: ask again soon
)

type Score struct {
	Value float64 `json:"value" doc:"IMDb: out of 10. Metacritic and Rotten Tomatoes: out of 100"`
	Votes int     `json:"votes,omitempty"`
}

type MovieScores struct {
	IMDb           *Score `json:"imdb,omitempty"`
	IMDbURL        string `json:"imdbUrl,omitempty"`
	Metacritic     *Score `json:"metacritic,omitempty"`
	RottenTomatoes *Score `json:"rottenTomatoes,omitempty" doc:"Critics score in percent"`
}

type entry struct {
	at     time.Time
	ttl    time.Duration
	scores MovieScores
}

type Service struct {
	store      *store.Store
	newServarr func(*store.ServarrInstance) *servarr.Client

	mu    sync.Mutex
	cache map[int]entry
}

func New(s *store.Store) *Service {
	return &Service{
		store: s, cache: map[int]entry{},
		// reading only: a dry-run client is the safe choice and changes nothing for reads
		newServarr: func(in *store.ServarrInstance) *servarr.Client { return servarr.New(in.Kind, in.URL, in.APIKey, true) },
	}
}

// Movie returns the scores Radarr knows for a TMDB movie. No Radarr, or no score, is an empty
// result, never an error: the page simply shows nothing.
func (s *Service) Movie(ctx context.Context, tmdbID int) MovieScores {
	s.mu.Lock()
	if e, ok := s.cache[tmdbID]; ok && time.Since(e.at) < e.ttl {
		s.mu.Unlock()
		return e.scores
	}
	s.mu.Unlock()

	scores, ok := s.lookup(ctx, tmdbID)
	ttl := hitTTL
	if !ok {
		ttl = missTTL
	}
	s.mu.Lock()
	if len(s.cache) > 5000 {
		s.cache = map[int]entry{}
	}
	s.cache[tmdbID] = entry{at: time.Now(), ttl: ttl, scores: scores}
	s.mu.Unlock()
	return scores
}

const (
	batchLimit    = 40
	batchWait     = 8 * time.Second // an answer is never held longer than this; slower lookups finish in the background
	batchParallel = 4
	lookupBudget  = 20 * time.Second
)

// Movies returns the scores of several movies at once (for the posters of a page). Scores that are
// not remembered yet are looked up a few at a time; whatever is not ready within a few seconds is
// left out of the answer, and keeps being fetched so the next request finds it.
func (s *Service) Movies(ctx context.Context, ids []int) map[int]MovieScores {
	out := map[int]MovieScores{}
	if len(ids) > batchLimit {
		ids = ids[:batchLimit]
	}
	var todo []int
	s.mu.Lock()
	for _, id := range ids {
		if e, ok := s.cache[id]; ok && time.Since(e.at) < e.ttl {
			out[id] = e.scores
		} else {
			todo = append(todo, id)
		}
	}
	s.mu.Unlock()
	if len(todo) == 0 {
		return out
	}

	type result struct {
		id     int
		scores MovieScores
	}
	results := make(chan result, len(todo))
	bg, cancel := context.WithTimeout(context.WithoutCancel(ctx), lookupBudget)
	go func() {
		defer cancel()
		sem := make(chan struct{}, batchParallel)
		var wg sync.WaitGroup
		for _, id := range todo {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				results <- result{id, s.Movie(bg, id)}
			}()
		}
		wg.Wait()
	}()
	timeout := time.After(batchWait)
	for range todo {
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
	delete(s.cache, tmdbID)
	s.mu.Unlock()
}

func (s *Service) lookup(ctx context.Context, tmdbID int) (MovieScores, bool) {
	inst, err := s.store.DefaultServarr(ctx, "radarr")
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			slog.Debug("ratings: no Radarr instance", "err", err)
		}
		return MovieScores{}, false
	}
	r, err := s.newServarr(inst).RatingsByTMDB(ctx, tmdbID)
	if err != nil {
		slog.Debug("ratings: Radarr lookup failed", "err", err)
		return MovieScores{}, false
	}
	var out MovieScores
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
	return out, out.IMDb != nil || out.Metacritic != nil || out.RottenTomatoes != nil
}
