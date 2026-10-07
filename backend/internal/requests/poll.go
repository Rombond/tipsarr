package requests

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Poll checks every request that was really sent to Radarr/Sonarr: it records download
// progress, and marks requests available when the files arrive. It does only reads, so it
// is safe in dry-run mode (and in dry-run no request is ever "sent", so there is nothing to poll).
// It returns the number of in-flight requests it looked at.
func (s *Service) Poll(ctx context.Context) (int, error) {
	inflight, err := s.store.InFlightRequests(ctx)
	if err != nil || len(inflight) == 0 {
		return 0, err
	}
	byInstance := map[string][]store.Request{}
	for _, r := range inflight {
		byInstance[r.InstanceID] = append(byInstance[r.InstanceID], r)
	}
	var firstErr error
	for instID, reqs := range byInstance {
		inst, err := s.store.GetServarr(ctx, instID)
		if err != nil {
			continue // instance was deleted; leave the requests as they are
		}
		client := s.newServarr(inst)
		queue, err := client.Queue(ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s queue: %w", inst.Name, err)
			}
			continue
		}
		byMedia := servarr.AggregateQueue(queue)
		for i := range reqs {
			if err := s.pollOne(ctx, client, &reqs[i], byMedia); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return len(inflight), firstErr
}

func (s *Service) pollOne(ctx context.Context, c *servarr.Client, r *store.Request, queue map[int]servarr.QueueItem) error {
	done := false
	var err error
	if r.MediaType == "movie" {
		var m *servarr.Movie
		if m, err = c.Movie(ctx, int(r.ServarrID)); err == nil {
			done = m.HasFile
		}
	} else {
		var sr *servarr.Series
		if sr, err = c.Series(ctx, int(r.ServarrID)); err == nil {
			done = sr.Complete()
		}
	}
	if err != nil {
		if servarr.IsNotFound(err) { // removed from Radarr/Sonarr behind our back
			r.Status, r.Error = store.StatusFailed, "removed from Radarr/Sonarr"
			s.clearProgress(r.ID)
			if uerr := s.store.UpdateRequest(ctx, r); uerr != nil {
				return uerr
			}
			s.changed(ctx, r)
			s.notify.Dispatch(notify.RequestFailed, s.payload(ctx, r))
			return nil
		}
		slog.Warn("poll request", "id", r.ID, "err", err)
		return nil // transient; try again next tick
	}

	if done {
		r.Status = store.StatusAvailable
		s.clearProgress(r.ID)
		if err := s.store.UpdateRequest(ctx, r); err != nil {
			return err
		}
		s.changed(ctx, r)
		s.hub.Publish("media.available", r.RequestedBy, map[string]any{"type": r.MediaType, "tmdbId": r.TMDBID}, false)
		s.notify.Dispatch(notify.MediaAvailable, s.payload(ctx, r))
		return nil
	}

	q, downloading := queue[int(r.ServarrID)]
	s.mu.Lock()
	prev, had := s.progress[r.ID]
	if downloading {
		s.progress[r.ID] = Progress{Percent: q.Percent(), ETASeconds: q.ETASeconds()}
	} else {
		delete(s.progress, r.ID)
	}
	cur := s.progress[r.ID]
	s.mu.Unlock()
	if downloading && (!had || prev != cur) {
		s.hub.Publish("request.progress", r.RequestedBy, map[string]any{"id": r.ID, "percent": cur.Percent, "etaSeconds": cur.ETASeconds}, false)
		if !had {
			s.changed(ctx, r) // searching -> downloading
		}
	} else if !downloading && had {
		s.changed(ctx, r) // download finished or paused: back to searching until files appear
	}
	return nil
}

func (s *Service) clearProgress(id string) {
	s.mu.Lock()
	delete(s.progress, id)
	s.mu.Unlock()
}
