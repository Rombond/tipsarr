package requests

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

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
		byMedia := map[int][]servarr.QueueItem{}
		for _, q := range queue {
			byMedia[q.MediaID] = append(byMedia[q.MediaID], q)
		}
		for i := range reqs {
			if err := s.pollOne(ctx, client, &reqs[i], byMedia); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return len(inflight), firstErr
}

func (s *Service) pollOne(ctx context.Context, c *servarr.Client, r *store.Request, queue map[int][]servarr.QueueItem) error {
	done := false
	recs := queue[int(r.ServarrID)]
	total := combine(recs) // percent / ETA of what the download client is transferring
	var seasons []SeasonProgress
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
			var pct int
			pct, seasons = showProgress(sr, recs)
			if len(recs) > 0 {
				total.Percent = pct
			}
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

	// "downloading" means the download client is really transferring it; a show with only some
	// episodes on disk, or still looking for releases, is "searching".
	downloading := false
	for _, q := range recs {
		downloading = downloading || q.Downloading()
	}
	s.mu.Lock()
	prev, had := s.progress[r.ID]
	if downloading {
		s.progress[r.ID] = total
	} else {
		delete(s.progress, r.ID)
	}
	cur := s.progress[r.ID]
	prevSeasons := s.seasons[r.ID]
	if r.MediaType == "tv" {
		s.seasons[r.ID] = seasons
	}
	s.mu.Unlock()
	seasonsChanged := r.MediaType == "tv" && !slices.Equal(prevSeasons, seasons)
	if downloading && (!had || prev != cur) {
		s.hub.Publish("request.progress", r.RequestedBy, map[string]any{"id": r.ID, "percent": cur.Percent, "etaSeconds": cur.ETASeconds}, false)
		if !had {
			s.changed(ctx, r) // searching -> downloading
		}
	} else if !downloading && had {
		s.changed(ctx, r) // download finished or paused: back to searching until files appear
	}
	if seasonsChanged && !(downloading && !had) {
		s.changed(ctx, r) // clients refetch the per-season percentages
	}
	return nil
}

// combine adds up the queue records of one title: sizes sum, the ETA is the longest.
func combine(recs []servarr.QueueItem) Progress {
	var size, left float64
	eta := 0
	for _, q := range recs {
		size += q.Size
		left += q.SizeLeft
		eta = max(eta, q.ETASeconds())
	}
	if size <= 0 {
		return Progress{}
	}
	return Progress{Percent: int((size - left) / size * 100), ETASeconds: eta}
}

// showProgress works out completion per season and overall from what Sonarr already has plus
// what the download client has transferred so far. Sonarr lists one queue record per episode
// (a season pack repeats its size on every episode), so each record counts as one episode
// completed to its own fraction.
func showProgress(sr *servarr.Series, recs []servarr.QueueItem) (int, []SeasonProgress) {
	frac := map[int]float64{}
	var fracAll float64
	for _, q := range recs {
		if q.Size <= 0 {
			continue
		}
		f := 1 - q.SizeLeft/q.Size
		frac[q.Season] += f
		fracAll += f
	}
	var out []SeasonProgress
	for _, sn := range sr.Seasons {
		n := sn.Statistics.EpisodeCount
		if sn.SeasonNumber <= 0 || n <= 0 {
			continue
		}
		have := float64(sn.Statistics.EpisodeFileCount) + frac[sn.SeasonNumber]
		pct := min(100, int(have*100/float64(n)))
		if sn.Statistics.EpisodeFileCount < n {
			pct = min(99, pct)
		}
		out = append(out, SeasonProgress{Season: sn.SeasonNumber, Percent: pct})
	}
	total := 0
	if n := sr.Statistics.EpisodeCount; n > 0 {
		total = min(99, int((float64(sr.Statistics.EpisodeFileCount)+fracAll)*100/float64(n)))
	}
	return total, out
}

func (s *Service) clearProgress(id string) {
	s.mu.Lock()
	delete(s.progress, id)
	delete(s.seasons, id)
	s.mu.Unlock()
}
