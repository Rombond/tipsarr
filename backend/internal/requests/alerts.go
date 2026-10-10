package requests

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// LibraryRefresher asks the Jellyfin library sync to run soon (debounced), so a finished download
// shows as available in Tipsarr.
type LibraryRefresher interface{ QueueLibrary() }

// How long an alert waits for the library to show the title, at most. Radarr and Sonarr have the
// file before Jellyfin has scanned it and before Tipsarr's library sync has seen it; the person
// who taps the alert expects the title to be playable.
const (
	titleHold  = 10 * time.Minute // movie or whole show: sent earlier as soon as the library has it
	seasonHold = 3 * time.Minute  // one season of a show: the library cannot tell us, so a short wait
)

// SetLibrary lets the service ask the library sync to refresh when a download finishes.
func (s *Service) SetLibrary(l LibraryRefresher) { s.lib = l }

// SetClock and SetAlertHold are for tests.
func (s *Service) SetClock(now func() time.Time) { s.now = now }
func (s *Service) SetAlertHold(title, season time.Duration) {
	s.titleHold, s.seasonHold = title, season
}

func (s *Service) refreshLibrary() {
	if s.lib != nil {
		s.lib.QueueLibrary()
	}
}

// noteSeasons stamps the requested seasons of a show that are complete, so each can be announced
// before the whole request is. Only requests with more than one season get season alerts.
func (s *Service) noteSeasons(ctx context.Context, r *store.Request, progress []SeasonProgress) {
	requested, err := s.store.RequestSeasons(ctx, []string{r.ID})
	if err != nil || len(requested[r.ID]) < 2 {
		return
	}
	want := map[int]bool{}
	for _, n := range requested[r.ID] {
		want[n] = true
	}
	for _, p := range progress {
		if !want[p.Season] || p.Percent < 100 {
			continue
		}
		if first, err := s.store.MarkSeasonDone(ctx, r.ID, p.Season, s.now().Unix()); err == nil && first {
			s.refreshLibrary()
		}
	}
}

// SendDueAlerts sends the "now available" alerts that were held back: for a finished request once the
// library shows the title (or after titleHold), for a finished season of a show still downloading after
// seasonHold. Each alert is sent once. The poll job runs it on every tick.
func (s *Service) SendDueAlerts(ctx context.Context) error {
	now := s.now().Unix()
	reqs, err := s.store.RequestsAwaitingAlert(ctx)
	if err != nil {
		return err
	}
	for i := range reqs {
		r := &reqs[i]
		if now-r.AvailableAt < int64(s.titleHold.Seconds()) && !s.inLibrary(ctx, r) {
			continue
		}
		skip := false
		if r.MediaType == "tv" {
			rows, err := s.store.RequestSeasonRows(ctx, r.ID)
			if err != nil {
				slog.Warn("alert seasons", "id", r.ID, "err", err)
				continue
			}
			var unannounced []int
			for _, row := range rows {
				if row.NotifiedAt == 0 {
					unannounced = append(unannounced, row.SeasonNumber)
				}
			}
			// every season of a multi-season request was already announced: the last alert would repeat the news
			skip = len(rows) > 1 && len(unannounced) == 0
			if err := s.store.MarkSeasonsAlerted(ctx, r.ID, unannounced, now); err != nil {
				slog.Warn("alert seasons", "id", r.ID, "err", err)
				continue
			}
		}
		if err := s.store.MarkRequestAlerted(ctx, r.ID, now); err != nil {
			slog.Warn("alert mark", "id", r.ID, "err", err)
			continue
		}
		if skip {
			continue
		}
		s.notify.Dispatch(notify.MediaAvailable, s.payload(ctx, r))
		s.notify.Push(notify.PushEvent{Name: "request.available", ID: r.ID, Subject: r.Title, Users: []string{r.RequestedBy}})
	}

	seasons, err := s.store.SeasonsAwaitingAlert(ctx)
	if err != nil {
		return err
	}
	for _, sn := range seasons {
		if now-sn.DoneAt < int64(s.seasonHold.Seconds()) {
			continue
		}
		r, err := s.store.GetRequest(ctx, sn.RequestID)
		if err != nil {
			continue
		}
		if err := s.store.MarkSeasonsAlerted(ctx, sn.RequestID, []int{sn.SeasonNumber}, now); err != nil {
			continue
		}
		s.notify.Push(notify.PushEvent{Name: "request.available", ID: r.ID, Subject: fmt.Sprintf("%s (S%d)", r.Title, sn.SeasonNumber), Users: []string{r.RequestedBy}})
	}
	return nil
}

// inLibrary reports whether the synced library already holds what was requested.
func (s *Service) inLibrary(ctx context.Context, r *store.Request) bool {
	if r.MediaType == "movie" {
		found, err := s.store.LibraryIDs(ctx, "movie", []int{int(r.TMDBID)})
		return err == nil && found[int(r.TMDBID)]
	}
	have, err := s.store.LibrarySeasons(ctx, int(r.TMDBID))
	if err != nil {
		return false
	}
	requested, err := s.store.RequestSeasons(ctx, []string{r.ID})
	if err != nil || len(requested[r.ID]) == 0 {
		return len(have) > 0
	}
	for _, n := range requested[r.ID] {
		if have[n] == 0 {
			return false
		}
	}
	return true
}
