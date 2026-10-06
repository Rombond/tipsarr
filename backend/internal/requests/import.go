package requests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// SettingAutoImport turns the periodic import on or off ("false" = off, anything else = on).
const SettingAutoImport = "servarr.auto_import"

// maxImportPerRun bounds the TMDB lookups of one run; the rest follows on the next run.
const maxImportPerRun = 60

// ImportFromServarr mirrors what Radarr/Sonarr already monitor but have not downloaded as
// "approved" requests, so Requests shows (and tracks) everything that is on its way, whoever
// added it. It only reads from Radarr/Sonarr and writes to Tipsarr's own database: nothing is
// requested or added anywhere, so it is safe in dry-run mode.
func (s *Service) ImportFromServarr(ctx context.Context) (string, error) {
	if v, _ := s.store.GetSetting(ctx, SettingAutoImport); v == "false" {
		return "auto-import is off", nil
	}
	instances, err := s.store.ListServarr(ctx)
	if err != nil {
		return "", err
	}
	imported, looked := 0, 0
	var firstErr error
	for i := range instances {
		inst := &instances[i]
		if inst.IsDefault != 1 {
			continue // the default instance is the one requests go to
		}
		c := s.newServarr(inst)
		var missing []candidate
		if inst.Kind == servarr.KindRadarr {
			movies, err := c.Movies(ctx)
			if err != nil {
				firstErr = fmt.Errorf("%s: %w", inst.Name, err)
				continue
			}
			for _, m := range movies {
				if m.Monitored && !m.HasFile && m.TMDBID > 0 {
					missing = append(missing, candidate{mediaType: "movie", tmdbID: m.TMDBID, servarrID: m.ID, title: m.Title})
				}
			}
		} else {
			series, err := c.AllSeries(ctx)
			if err != nil {
				firstErr = fmt.Errorf("%s: %w", inst.Name, err)
				continue
			}
			for _, sr := range series {
				if !sr.Monitored || sr.TMDBID == 0 {
					continue
				}
				if seasons := sr.MissingSeasons(); len(seasons) > 0 {
					missing = append(missing, candidate{mediaType: "tv", tmdbID: sr.TMDBID, servarrID: sr.ID, title: sr.Title, seasons: seasons})
				}
			}
		}
		sort.Slice(missing, func(a, b int) bool { return missing[a].tmdbID < missing[b].tmdbID })
		n, seen, err := s.importCandidates(ctx, inst, missing, maxImportPerRun-looked)
		imported, looked = imported+n, looked+seen
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return fmt.Sprintf("%d imported", imported), firstErr
}

type candidate struct {
	mediaType string
	tmdbID    int
	servarrID int
	title     string
	seasons   []int
}

func (s *Service) importCandidates(ctx context.Context, inst *store.ServarrInstance, cands []candidate, budget int) (imported, looked int, err error) {
	byType := map[string][]int{}
	for _, c := range cands {
		byType[c.mediaType] = append(byType[c.mediaType], c.tmdbID)
	}
	have := map[string]map[int]string{}
	for t, ids := range byType {
		if have[t], err = s.store.ActiveRequestStatuses(ctx, t, ids); err != nil {
			return 0, 0, err
		}
	}
	for _, c := range cands {
		if have[c.mediaType][c.tmdbID] != "" {
			continue // already tracked (pending or approved)
		}
		if looked >= budget {
			break
		}
		looked++
		title, poster, release := c.title, "", ""
		d, derr := s.media.Detail(ctx, media.Opts{}, c.mediaType, c.tmdbID)
		switch {
		case derr == nil:
			if d.Availability == media.AvailabilityAvailable {
				continue // it is in the Jellyfin library already
			}
			title, poster, release = d.Title, d.PosterPath, d.ReleaseDate
		case errors.Is(derr, media.ErrNotConfigured):
			// no TMDB key: keep the title Radarr/Sonarr gave us
		default:
			slog.Warn("import: TMDB lookup failed", "title", c.title, "err", derr)
			continue
		}
		r := &store.Request{
			ID: store.NewID(), MediaType: c.mediaType, TMDBID: int64(c.tmdbID), Title: title, PosterPath: poster, ReleaseDate: release,
			Status: store.StatusApproved, InstanceID: inst.ID, ServarrID: int64(c.servarrID), SentAt: nowUnix(), Source: inst.Kind,
		}
		if err := s.store.CreateRequest(ctx, r, c.seasons); err != nil {
			return imported, looked, err
		}
		s.changed(ctx, r)
		imported++
	}
	return imported, looked, nil
}
