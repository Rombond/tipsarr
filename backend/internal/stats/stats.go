// Package stats builds the numbers of the Stats page from what Tipsarr has: the plays copied from
// Jellyfin's Playback Reporting plugin when there are any (exact), otherwise the synced watch
// history combined with the library's runtimes (estimated).
package stats

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/playback"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	minPlaySeconds = 60 // shorter sessions are skips, not plays
	topGenres      = 12
	topTitles      = 10
)

type Service struct{ store *store.Store }

func New(s *store.Store) *Service { return &Service{store: s} }

type StatsBucket struct {
	Name   string  `json:"name"`
	Hours  float64 `json:"hours"`
	Titles int     `json:"titles"`
}

type StatsMonth struct {
	Month string  `json:"month" doc:"YYYY-MM"`
	Hours float64 `json:"hours"`
	Plays int     `json:"plays"`
}

type StatsTop struct {
	Type      string  `json:"type" enum:"movie,tv"`
	TMDBID    int64   `json:"tmdbId"`
	Title     string  `json:"title"`
	Plays     int     `json:"plays"`
	Hours     float64 `json:"hours"`
	PosterURL string  `json:"posterUrl,omitempty"`
}

type StatsTotals struct {
	Titles int     `json:"titles"`
	Movies int     `json:"movies"`
	Shows  int     `json:"shows"`
	Plays  int     `json:"plays"`
	Hours  float64 `json:"hours"`
}

type StatsRequests struct {
	Made      int `json:"made"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Available int `json:"available"`
	Declined  int `json:"declined"`
	Failed    int `json:"failed"`
}

type StatsPlugin struct {
	Hint bool `json:"hint" doc:"Admins only: the Playback Reporting plugin is missing and would make these numbers exact"`
}

type StatsReport struct {
	Source   string        `json:"source" enum:"plugin,estimate" doc:"plugin = exact plays from Jellyfin's Playback Reporting; estimate = watch history x runtime"`
	User     string        `json:"user" doc:"A user id, or all"`
	Period   string        `json:"period" enum:"30d,12m,all"`
	Totals   StatsTotals   `json:"totals"`
	Genres   []StatsBucket `json:"genres"`
	Decades  []StatsBucket `json:"decades"`
	Months   []StatsMonth  `json:"months" doc:"Last 12 months, oldest first (plugin source only)"`
	Weekdays []float64     `json:"weekdays" doc:"Hours per weekday, Monday first (plugin source only)"`
	Hours24  []float64     `json:"hoursOfDay" doc:"Hours per hour of the day (plugin source only)"`
	Top      []StatsTop    `json:"top"`
	Requests StatsRequests `json:"requests"`
	Plugin   StatsPlugin   `json:"plugin"`
}

type title struct {
	typ    string
	tmdb   int64
	name   string
	jf     string
	plays  int
	secs   float64
	genres []string
	year   int
	tag    string
}

// Compute returns the stats of one user ("" = everyone) for a period (30d, 12m or all).
func (s *Service) Compute(ctx context.Context, userID, period string, adminViewer bool) (*StatsReport, error) {
	now := time.Now()
	var since time.Time
	switch period {
	case "30d":
		since = now.AddDate(0, 0, -30)
	case "12m":
		since = now.AddDate(-1, 0, 0)
	default:
		period = "all"
	}
	lib, err := s.store.AllLibrary(ctx)
	if err != nil {
		return nil, err
	}
	meta := map[string]store.LibraryItem{}
	for _, it := range lib {
		meta[fmt.Sprintf("%s:%d", it.MediaType, it.TMDBID)] = it
	}

	res := &StatsReport{User: userID, Period: period, Genres: []StatsBucket{}, Decades: []StatsBucket{}, Months: []StatsMonth{}, Weekdays: []float64{}, Hours24: []float64{}, Top: []StatsTop{}}
	if res.User == "" {
		res.User = "all"
	}
	events, err := s.store.CountWatchEvents(ctx)
	if err != nil {
		return nil, err
	}
	titles := map[string]*title{}
	if events > 0 {
		res.Source = "plugin"
		err = s.fromPlugin(ctx, res, userID, since, now, meta, titles)
	} else {
		res.Source = "estimate"
		err = s.fromHistory(ctx, userID, meta, titles)
	}
	if err != nil {
		return nil, err
	}
	aggregate(res, titles)

	st, err := s.store.RequestsByStatus(ctx, userID, since.Unix()*b2i(!since.IsZero()))
	if err != nil {
		return nil, err
	}
	for k, n := range st {
		res.Requests.Made += n
		switch k {
		case store.StatusPending:
			res.Requests.Pending = n
		case store.StatusApproved:
			res.Requests.Approved = n
		case store.StatusAvailable:
			res.Requests.Available = n
		case store.StatusDeclined:
			res.Requests.Declined = n
		case store.StatusFailed:
			res.Requests.Failed = n
		}
	}
	if adminViewer && res.Source == "estimate" {
		state, _ := s.store.GetSetting(ctx, playback.SettingPlugin)
		res.Plugin.Hint = state == "missing"
	}
	return res, nil
}

func b2i(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func key(typ string, tmdb int64, jf, name string) string {
	if tmdb > 0 {
		return typ + ":" + strconv.FormatInt(tmdb, 10)
	}
	return typ + ":?" + jf + name
}

func (s *Service) fromPlugin(ctx context.Context, res *StatsReport, userID string, since, now time.Time, meta map[string]store.LibraryItem, titles map[string]*title) error {
	rows, err := s.store.WatchEvents(ctx, userID, since.Unix())
	if err != nil {
		return err
	}
	// timeline buckets: the last 12 calendar months, oldest first
	monthIdx := map[string]int{}
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -11, 0)
	for i := 0; i < 12; i++ {
		m := first.AddDate(0, i, 0).Format("2006-01")
		monthIdx[m] = i
		res.Months = append(res.Months, StatsMonth{Month: m})
	}
	res.Weekdays, res.Hours24 = make([]float64, 7), make([]float64, 24)
	for _, e := range rows {
		if e.Seconds < minPlaySeconds {
			continue
		}
		t := time.Unix(e.PlayedAt, 0)
		hours := float64(e.Seconds) / 3600
		if i, ok := monthIdx[t.Format("2006-01")]; ok {
			res.Months[i].Hours += hours
			res.Months[i].Plays++
		}
		res.Weekdays[(int(t.Weekday())+6)%7] += hours // Monday first
		res.Hours24[t.Hour()] += hours

		k := key(e.MediaType, e.TMDBID, e.JellyfinID, e.Title)
		ti := titles[k]
		if ti == nil {
			ti = &title{typ: e.MediaType, tmdb: e.TMDBID, name: e.Title, jf: e.JellyfinID}
			if it, ok := meta[fmt.Sprintf("%s:%d", e.MediaType, e.TMDBID)]; ok {
				fill(ti, it)
			}
			titles[k] = ti
		}
		ti.plays++
		ti.secs += float64(e.Seconds)
	}
	return nil
}

func fill(t *title, it store.LibraryItem) {
	t.name, t.jf, t.year, t.tag = it.Title, it.JellyfinID, it.Year, it.ImageTag
	if g := strings.Trim(it.Genres, "|"); g != "" {
		t.genres = strings.Split(g, "|")
	}
}

// fromHistory estimates from the synced history: plays x the library's runtime (a movie's length,
// or a show's typical episode length).
func (s *Service) fromHistory(ctx context.Context, userID string, meta map[string]store.LibraryItem, titles map[string]*title) error {
	rows, err := s.store.WatchHistoryOf(ctx, userID)
	if err != nil {
		return err
	}
	for _, h := range rows {
		it, inLib := meta[fmt.Sprintf("%s:%d", h.MediaType, h.TMDBID)]
		k := key(h.MediaType, h.TMDBID, "", "")
		ti := titles[k]
		if ti == nil {
			ti = &title{typ: h.MediaType, tmdb: h.TMDBID}
			if inLib {
				fill(ti, it)
			} else {
				ti.name = fmt.Sprintf("#%d", h.TMDBID)
			}
			titles[k] = ti
		}
		ti.plays += max(h.PlayCount, 1)
		if inLib {
			ti.secs += float64(max(h.PlayCount, 1)) * float64(it.RuntimeMin) * 60
		}
	}
	return nil
}

func aggregate(res *StatsReport, titles map[string]*title) {
	genre := map[string]*StatsBucket{}
	decade := map[string]*StatsBucket{}
	var all []*title
	for _, t := range titles {
		all = append(all, t)
		h := t.secs / 3600
		res.Totals.Titles++
		res.Totals.Plays += t.plays
		res.Totals.Hours += h
		if t.typ == "movie" {
			res.Totals.Movies++
		} else {
			res.Totals.Shows++
		}
		for _, g := range t.genres {
			b := genre[g]
			if b == nil {
				b = &StatsBucket{Name: g}
				genre[g] = b
			}
			b.Hours += h
			b.Titles++
		}
		if t.year > 0 {
			d := strconv.Itoa(t.year/10*10) + "s"
			b := decade[d]
			if b == nil {
				b = &StatsBucket{Name: d}
				decade[d] = b
			}
			b.Hours += h
			b.Titles++
		}
	}
	for _, b := range genre {
		res.Genres = append(res.Genres, *b)
	}
	sort.Slice(res.Genres, func(i, j int) bool {
		a, b := res.Genres[i], res.Genres[j]
		if a.Hours != b.Hours {
			return a.Hours > b.Hours
		}
		if a.Titles != b.Titles {
			return a.Titles > b.Titles
		}
		return a.Name < b.Name
	})
	if len(res.Genres) > topGenres {
		res.Genres = res.Genres[:topGenres]
	}
	for _, b := range decade {
		res.Decades = append(res.Decades, *b)
	}
	sort.Slice(res.Decades, func(i, j int) bool { return res.Decades[i].Name < res.Decades[j].Name })

	sort.Slice(all, func(i, j int) bool {
		if all[i].plays != all[j].plays {
			return all[i].plays > all[j].plays
		}
		if all[i].secs != all[j].secs {
			return all[i].secs > all[j].secs
		}
		return all[i].name < all[j].name
	})
	for _, t := range all {
		if len(res.Top) == topTitles {
			break
		}
		if t.tmdb == 0 {
			continue // cannot link to a details page
		}
		top := StatsTop{Type: t.typ, TMDBID: t.tmdb, Title: t.name, Plays: t.plays, Hours: t.secs / 3600}
		if t.tag != "" && t.jf != "" {
			top.PosterURL = "/api/v1/images/jellyfin/" + t.jf + "?tag=" + t.tag
		}
		res.Top = append(res.Top, top)
	}
}
