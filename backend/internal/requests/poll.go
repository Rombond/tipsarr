package requests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Poll checks every request that was really sent to Radarr/Sonarr: it records download
// progress, and marks requests available when the files arrive. It does only reads, so it
// is safe in dry-run mode (and in dry-run no request is ever "sent", so there is nothing to poll).
// It returns the number of in-flight requests it looked at.
func (s *Service) Poll(ctx context.Context) (int, error) {
	if err := s.CloseFulfilled(ctx); err != nil {
		slog.Warn("close fulfilled requests", "err", err)
	}
	if err := s.SendDueAlerts(ctx); err != nil {
		slog.Warn("send due alerts", "err", err)
	}
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
		// The instance only reads its download clients once a minute; ask for a fresh look now so
		// the next tick (15 s) sees real progress instead of a minute-old queue. Best effort.
		if err := client.RefreshDownloads(ctx); err != nil && !errors.Is(err, servarr.ErrDryRun) {
			slog.Debug("refresh downloads", "instance", inst.Name, "err", err)
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
			s.record(ctx, r.ID, "failed", "", r.Error)
			s.changed(ctx, r)
			s.notify.Send(s.event(ctx, notify.RequestFailed, r, ""))
			s.notify.Push(notify.PushEvent{Name: "request.failed", ID: r.ID, Subject: r.Title, Users: []string{r.RequestedBy}})
			return nil
		}
		slog.Warn("poll request", "id", r.ID, "err", err)
		return nil // transient; try again next tick
	}

	if done {
		if err := s.markAvailable(ctx, r); err != nil {
			return err
		}
		s.refreshLibrary()
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
	if r.MediaType == "tv" {
		s.noteSeasons(ctx, r, seasons)
	}
	seasonsChanged := r.MediaType == "tv" && !slices.Equal(prevSeasons, seasons)
	if downloading && (!had || prev != cur) {
		s.hub.Publish("request.progress", r.RequestedBy, map[string]any{"id": r.ID, "percent": cur.Percent, "etaSeconds": cur.ETASeconds}, false)
		if !had {
			// the first time only: after a restart the progress is forgotten and the transfer looks new again
			if seen, err := s.store.HasRequestEvent(ctx, r.ID, "downloading"); err == nil && !seen {
				s.record(ctx, r.ID, "downloading", "", "")
			}
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

// markAvailable turns a request available. The webhook and the push wait in SendDueAlerts until the
// library shows the title.
func (s *Service) markAvailable(ctx context.Context, r *store.Request) error {
	r.Status = store.StatusAvailable
	r.AvailableAt = s.now().Unix()
	s.clearProgress(r.ID)
	if err := s.store.UpdateRequest(ctx, r); err != nil {
		return err
	}
	s.announceAvailable(ctx, r)
	return nil
}

func (s *Service) announceAvailable(ctx context.Context, r *store.Request) {
	s.record(ctx, r.ID, "available", "", "")
	s.changed(ctx, r)
	s.hub.Publish("media.available", r.RequestedBy, map[string]any{"type": r.MediaType, "tmdbId": r.TMDBID}, false)
}

// CloseFulfilled marks available every pending or approved-but-unsent request whose title the synced
// library already holds completely, e.g. because it was added outside Tipsarr. Requests sent to
// Radarr/Sonarr are left to pollOne. The requester gets the usual "now available" alert
// (SendDueAlerts sends it at once, the library has the title). It only reads, never writes to
// Radarr/Sonarr, and runs only when the library snapshot changed.
func (s *Service) CloseFulfilled(ctx context.Context) error {
	gen := s.store.LibraryGeneration()
	s.mu.Lock()
	seen := s.libGenSeen
	s.mu.Unlock()
	if seen == gen {
		return nil
	}
	open, err := s.store.UnsentOpenRequests(ctx)
	if err != nil {
		return err
	}
	var movieIDs, showIDs []int
	var tvReqs []string
	for _, r := range open {
		if r.MediaType == "movie" {
			movieIDs = append(movieIDs, int(r.TMDBID))
		} else {
			showIDs = append(showIDs, int(r.TMDBID))
			tvReqs = append(tvReqs, r.ID)
		}
	}
	movies, err := s.store.LibraryIDs(ctx, "movie", movieIDs)
	if err != nil {
		return err
	}
	shows, err := s.store.LibrarySeasonsOf(ctx, showIDs)
	if err != nil {
		return err
	}
	requested, err := s.store.RequestSeasons(ctx, tvReqs)
	if err != nil {
		return err
	}
	for i := range open {
		r := &open[i]
		if r.MediaType == "movie" {
			if !movies[int(r.TMDBID)] {
				continue
			}
		} else if !s.showComplete(ctx, r, requested[r.ID], shows[int(r.TMDBID)]) {
			continue
		}
		ok, err := s.store.MarkRequestAvailable(ctx, r.ID, s.now().Unix())
		if err != nil {
			return err
		}
		if !ok {
			continue // approved, declined or deleted meanwhile
		}
		r.Status, r.AvailableAt = store.StatusAvailable, s.now().Unix()
		s.clearProgress(r.ID)
		s.announceAvailable(ctx, r)
	}
	s.mu.Lock()
	s.libGenSeen = gen
	s.mu.Unlock()
	return nil
}

// showComplete reports whether the library holds every episode of every requested season. The
// library counts seasons in TVDB numbering, so requests numbered otherwise are skipped. Same rule as
// Create: a season is complete when the library has all its episodes (media.Season.Complete).
func (s *Service) showComplete(ctx context.Context, r *store.Request, want []int, have map[int]int) bool {
	if len(want) == 0 || r.SeasonNumbering != media.NumberingTVDB || s.media == nil {
		return false
	}
	for _, n := range want {
		if have[n] == 0 {
			return false // cheap check first: no media.Detail call unless the show looks present
		}
	}
	d, err := s.media.Detail(ctx, media.Opts{Seasons: true}, "tv", int(r.TMDBID))
	if err != nil || d.SeasonNumbering != media.NumberingTVDB {
		return false
	}
	complete := map[int]bool{}
	for _, sn := range d.Seasons {
		complete[sn.Number] = sn.Complete()
	}
	for _, n := range want {
		if !complete[n] {
			return false
		}
	}
	return true
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
