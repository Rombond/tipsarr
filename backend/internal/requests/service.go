// Package requests implements the request lifecycle:
//
//	pending -> approved -> (searching -> downloading ->) available
//	        \-> declined            \-> failed (retry)
//
// Nothing is ever requested automatically: every request comes from a user action.
// While dry-run is on, approving moves a request to "approved" without contacting
// Radarr/Sonarr for writes (the servarr client enforces this).
package requests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

var (
	ErrAlreadyAvailable = errors.New("already available in the library")
	ErrDuplicate        = errors.New("this title already has an active request")
	ErrNoInstance       = errors.New("no Radarr/Sonarr instance is configured for this media type")
	ErrBadState         = errors.New("request is not in a state that allows this")
	ErrForbidden        = errors.New("not allowed")
	ErrInvalid          = errors.New("invalid request")
)

type Service struct {
	store  *store.Store
	media  *media.Service
	hub    *events.Hub
	notify *notify.Service
	dryRun bool
	// newServarr builds a client for an instance; swapped in tests.
	newServarr func(in *store.ServarrInstance) *servarr.Client

	mu       sync.Mutex
	progress map[string]Progress
}

func New(s *store.Store, m *media.Service, h *events.Hub, n *notify.Service, dryRun bool) *Service {
	svc := &Service{store: s, media: m, hub: h, notify: n, dryRun: dryRun, progress: map[string]Progress{}}
	svc.newServarr = func(in *store.ServarrInstance) *servarr.Client {
		return servarr.New(in.Kind, in.URL, in.APIKey, dryRun)
	}
	return svc
}

type Progress struct {
	Percent    int `json:"percent"`
	ETASeconds int `json:"etaSeconds"`
}

type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// View is the API shape of a request.
type View struct {
	ID            string    `json:"id"`
	Type          string    `json:"type" enum:"movie,tv"`
	TMDBID        int       `json:"tmdbId"`
	Title         string    `json:"title"`
	PosterPath    string    `json:"posterPath,omitempty"`
	ReleaseDate   string    `json:"releaseDate,omitempty"`
	Seasons       []int     `json:"seasons,omitempty"`
	Status        string    `json:"status" enum:"pending,approved,declined,failed,available"`
	Stage         string    `json:"stage" enum:"requested,approved,searching,downloading,available,declined,failed" doc:"User-facing lifecycle step"`
	RequestedBy   UserRef   `json:"requestedBy"`
	DecidedBy     *UserRef  `json:"decidedBy,omitempty"`
	DeclineReason string    `json:"declineReason,omitempty"`
	DryRun        bool      `json:"dryRun" doc:"Approved in dry-run mode: nothing was sent to Radarr/Sonarr"`
	Error         string    `json:"error,omitempty"`
	Progress      *Progress `json:"progress,omitempty"`
	CreatedAt     int64     `json:"createdAt"`
	UpdatedAt     int64     `json:"updatedAt"`
}

func (s *Service) stage(r *store.Request) (string, *Progress) {
	switch r.Status {
	case store.StatusPending:
		return "requested", nil
	case store.StatusDeclined, store.StatusFailed, store.StatusAvailable:
		return r.Status, nil
	}
	if r.SentAt == 0 {
		return "approved", nil // dry-run: never sent
	}
	s.mu.Lock()
	p, ok := s.progress[r.ID]
	s.mu.Unlock()
	if ok {
		return "downloading", &p
	}
	return "searching", nil
}

func (s *Service) views(ctx context.Context, rows []store.Request) ([]View, error) {
	ids := make([]string, 0, len(rows))
	people := map[string]bool{}
	for _, r := range rows {
		ids = append(ids, r.ID)
		people[r.RequestedBy] = true
		if r.DecidedBy != "" {
			people[r.DecidedBy] = true
		}
	}
	pl := make([]string, 0, len(people))
	for id := range people {
		pl = append(pl, id)
	}
	names, err := s.store.UserNames(ctx, pl)
	if err != nil {
		return nil, err
	}
	seasons, err := s.store.RequestSeasons(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]View, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		stage, prog := s.stage(r)
		v := View{
			ID: r.ID, Type: r.MediaType, TMDBID: int(r.TMDBID), Title: r.Title, PosterPath: r.PosterPath,
			ReleaseDate: r.ReleaseDate, Seasons: seasons[r.ID], Status: r.Status, Stage: stage,
			RequestedBy: UserRef{ID: r.RequestedBy, Name: names[r.RequestedBy]}, DeclineReason: r.DeclineReason,
			DryRun: r.DryRun == 1, Error: r.Error, Progress: prog, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
		if r.DecidedBy != "" {
			v.DecidedBy = &UserRef{ID: r.DecidedBy, Name: names[r.DecidedBy]}
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) view(ctx context.Context, r *store.Request) (*View, error) {
	vs, err := s.views(ctx, []store.Request{*r})
	if err != nil {
		return nil, err
	}
	return &vs[0], nil
}

// ---- reads -----------------------------------------------------------------------------------

type ListParams struct {
	Filter     string // all | mine | pending | approved | available | declined | failed
	Take, Skip int
}

type ListResult struct {
	Total int
	Items []View
}

func canSee(u *store.User, r *store.Request) bool {
	return u.Role == store.RoleAdmin || r.RequestedBy == u.ID
}

func (s *Service) List(ctx context.Context, u *store.User, p ListParams) (*ListResult, error) {
	f := store.RequestFilter{Take: p.Take, Skip: p.Skip}
	if u.Role != store.RoleAdmin || p.Filter == "mine" {
		f.RequestedBy = u.ID
	}
	switch p.Filter {
	case "pending", "approved", "available", "declined", "failed":
		f.Statuses = []string{p.Filter}
	}
	rows, total, err := s.store.ListRequests(ctx, f)
	if err != nil {
		return nil, err
	}
	items, err := s.views(ctx, rows)
	for i := range items {
		redact(&items[i], u)
	}
	return &ListResult{Total: total, Items: items}, err
}

func (s *Service) Get(ctx context.Context, u *store.User, id string) (*View, error) {
	r, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canSee(u, r) {
		return nil, store.ErrNotFound // do not reveal other users' requests
	}
	v, err := s.view(ctx, r)
	if v != nil {
		redact(v, u)
	}
	return v, err
}

// redact hides technical failure details (internal URLs, upstream response bodies) from
// everyone but admins; webhooks pass a nil user and never carry them either.
func redact(v *View, u *store.User) {
	if v.Error != "" && (u == nil || u.Role != store.RoleAdmin) {
		v.Error = "Sending to the download manager failed; an admin can see the details."
	}
}

func (s *Service) Counts(ctx context.Context, u *store.User) (map[string]int, error) {
	uid := u.ID
	if u.Role == store.RoleAdmin {
		uid = ""
	}
	c, err := s.store.RequestCounts(ctx, uid)
	if err != nil {
		return nil, err
	}
	for _, k := range []string{store.StatusPending, store.StatusApproved, store.StatusAvailable, store.StatusDeclined, store.StatusFailed} {
		if _, ok := c[k]; !ok {
			c[k] = 0
		}
	}
	return c, nil
}

// ---- create ------------------------------------------------------------------------------------

type CreateParams struct {
	Type    string
	TMDBID  int
	Seasons []int // TV only; empty = every regular season
}

// Create records a request from a user. Admin requests are approved right away.
func (s *Service) Create(ctx context.Context, u *store.User, p CreateParams) (*View, error) {
	if p.Type != "movie" && p.Type != "tv" || p.TMDBID <= 0 {
		return nil, ErrInvalid
	}
	d, err := s.media.Detail(ctx, media.Opts{Language: u.Language, Region: u.Region}, p.Type, p.TMDBID)
	if err != nil {
		return nil, err // media.ErrNotFound / ErrNotConfigured pass through
	}
	if d.Availability == media.AvailabilityAvailable {
		return nil, ErrAlreadyAvailable
	}
	active, err := s.store.ActiveRequestStatuses(ctx, p.Type, []int{p.TMDBID})
	if err != nil {
		return nil, err
	}
	if active[p.TMDBID] != "" {
		return nil, ErrDuplicate
	}

	var seasons []int
	if p.Type == "tv" {
		valid := map[int]bool{}
		for _, sn := range d.Seasons {
			if sn.Number > 0 {
				valid[sn.Number] = true
			}
		}
		if len(valid) == 0 {
			return nil, fmt.Errorf("%w: this show has no seasons yet", ErrInvalid)
		}
		if len(p.Seasons) == 0 {
			for n := range valid {
				seasons = append(seasons, n)
			}
		} else {
			seen := map[int]bool{}
			for _, n := range p.Seasons {
				if !valid[n] {
					return nil, fmt.Errorf("%w: season %d does not exist", ErrInvalid, n)
				}
				if !seen[n] {
					seen[n] = true
					seasons = append(seasons, n)
				}
			}
		}
		sort.Ints(seasons)
	}

	r := &store.Request{
		ID: store.NewID(), MediaType: p.Type, TMDBID: int64(p.TMDBID), Title: d.Title, PosterPath: d.PosterPath,
		ReleaseDate: d.ReleaseDate, RequestedBy: u.ID, Status: store.StatusPending,
	}
	if err := s.store.CreateRequest(ctx, r, seasons); err != nil {
		return nil, err
	}
	s.changed(ctx, r)
	s.notify.Dispatch(notify.RequestCreated, s.payload(ctx, r))

	if u.Role == store.RoleAdmin {
		v, err := s.Approve(ctx, u, r.ID, nil)
		if errors.Is(err, ErrNoInstance) {
			return s.view(ctx, r) // stays pending until a Radarr/Sonarr instance is configured
		}
		return v, err
	}
	return s.view(ctx, r)
}

// ---- decisions ----------------------------------------------------------------------------------

type Overrides struct {
	ProfileID  *int
	RootFolder string
}

// Approve (or retry a failed request) and send it to Radarr/Sonarr unless dry-run is on.
func (s *Service) Approve(ctx context.Context, by *store.User, id string, ov *Overrides) (*View, error) {
	if by.Role != store.RoleAdmin {
		return nil, ErrForbidden
	}
	r, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Status != store.StatusPending && r.Status != store.StatusFailed {
		return nil, ErrBadState
	}
	kind := servarr.KindRadarr
	if r.MediaType == "tv" {
		kind = servarr.KindSonarr
	}
	inst, err := s.store.DefaultServarr(ctx, kind)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrNoInstance
	}
	if err != nil {
		return nil, err
	}
	profile, root := inst.QualityProfileID, inst.RootFolder
	explicitRoot := ov != nil && ov.RootFolder != ""
	if ov != nil {
		if ov.ProfileID != nil {
			profile = *ov.ProfileID
		}
		if ov.RootFolder != "" {
			root = ov.RootFolder
		}
	}
	if !explicitRoot && r.MediaType == "movie" && inst.GenreRoots != "" {
		if d, err := s.media.Detail(ctx, media.Opts{}, "movie", int(r.TMDBID)); err == nil {
			if gr := GenreRoot(inst.GenreRoots, d.Genres); gr != "" {
				root = gr
			}
		}
	}

	seriesType := ""
	if !explicitRoot && r.MediaType == "tv" && inst.AnimeRoot != "" {
		if d, err := s.media.Detail(ctx, media.Opts{}, "tv", int(r.TMDBID)); err == nil && IsAnime(d) {
			root, seriesType = inst.AnimeRoot, "anime"
		}
	}

	r.DecidedBy, r.InstanceID, r.DeclineReason, r.Error = by.ID, inst.ID, "", ""
	client := s.newServarr(inst)
	servarrID, sendErr := s.send(ctx, client, r, profile, root, seriesType)
	switch {
	case errors.Is(sendErr, servarr.ErrDryRun):
		r.Status, r.DryRun, r.SentAt, r.ServarrID = store.StatusApproved, 1, 0, 0
		slog.Info("dry-run: request approved but not sent", "title", r.Title, "type", r.MediaType)
	case sendErr != nil:
		r.Status, r.Error = store.StatusFailed, truncate(sendErr.Error(), 500)
	default:
		r.Status, r.DryRun, r.SentAt, r.ServarrID = store.StatusApproved, 0, time.Now().Unix(), int64(servarrID)
	}
	if err := s.store.UpdateRequest(ctx, r); err != nil {
		return nil, err
	}
	s.changed(ctx, r)
	if r.Status == store.StatusFailed {
		s.notify.Dispatch(notify.RequestFailed, s.payload(ctx, r))
	} else {
		s.notify.Dispatch(notify.RequestApproved, s.payload(ctx, r))
	}
	return s.view(ctx, r)
}

// IsAnime reports whether a show looks like anime: TMDB has no anime genre, so the rule is
// "Animation" (genre 16) with Japanese as the original language.
func IsAnime(d *media.Detail) bool {
	if d.OriginalLanguage != "ja" {
		return false
	}
	for _, g := range d.Genres {
		if g.ID == 16 {
			return true
		}
	}
	return false
}

// GenreRoot picks the root folder for a movie from the instance's genre map (JSON
// {"<tmdb genre id>": "<folder>"}): the first of the movie's genres that has a mapping wins.
func GenreRoot(genreRootsJSON string, genres []media.Genre) string {
	var m map[string]string
	if json.Unmarshal([]byte(genreRootsJSON), &m) != nil {
		return ""
	}
	for _, g := range genres {
		if p := m[strconv.Itoa(g.ID)]; p != "" {
			return p
		}
	}
	return ""
}

func (s *Service) send(ctx context.Context, c *servarr.Client, r *store.Request, profile int, root, seriesType string) (int, error) {
	if r.MediaType == "movie" {
		return c.AddMovie(ctx, int(r.TMDBID), profile, root)
	}
	d, err := s.media.Detail(ctx, media.Opts{}, "tv", int(r.TMDBID))
	if err != nil {
		return 0, err
	}
	if d.TVDBID == 0 {
		return 0, errors.New("TMDB has no TVDB id for this show, so Sonarr cannot add it")
	}
	seasons, err := s.store.RequestSeasons(ctx, []string{r.ID})
	if err != nil {
		return 0, err
	}
	return c.AddSeries(ctx, d.TVDBID, profile, root, seasons[r.ID], seriesType)
}

func (s *Service) Decline(ctx context.Context, by *store.User, id, reason string) (*View, error) {
	if by.Role != store.RoleAdmin {
		return nil, ErrForbidden
	}
	r, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.Status != store.StatusPending && r.Status != store.StatusFailed {
		return nil, ErrBadState
	}
	r.Status, r.DecidedBy, r.DeclineReason = store.StatusDeclined, by.ID, truncate(reason, 500)
	if err := s.store.UpdateRequest(ctx, r); err != nil {
		return nil, err
	}
	s.changed(ctx, r)
	s.notify.Dispatch(notify.RequestDeclined, s.payload(ctx, r))
	return s.view(ctx, r)
}

// Delete removes a request. Admins can delete any; owners only their unfinished ones.
func (s *Service) Delete(ctx context.Context, u *store.User, id string) error {
	r, err := s.store.GetRequest(ctx, id)
	if err != nil {
		return err
	}
	if !canSee(u, r) {
		return store.ErrNotFound
	}
	if u.Role != store.RoleAdmin && r.Status != store.StatusPending && r.Status != store.StatusDeclined && r.Status != store.StatusFailed {
		return ErrForbidden
	}
	if err := s.store.DeleteRequest(ctx, id); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.progress, id)
	s.mu.Unlock()
	s.hub.Publish("request.updated", r.RequestedBy, map[string]any{"id": id, "status": "deleted", "stage": "deleted"}, false)
	return nil
}

func (s *Service) Progress(ctx context.Context, u *store.User, id string) (*View, error) {
	return s.Get(ctx, u, id)
}

// ---- events / notifications ----------------------------------------------------------------

func (s *Service) changed(ctx context.Context, r *store.Request) {
	stage, _ := s.stage(r)
	s.hub.Publish("request.updated", r.RequestedBy, map[string]any{"id": r.ID, "status": r.Status, "stage": stage}, false)
}

type payload struct {
	Request View `json:"request"`
}

func (s *Service) payload(ctx context.Context, r *store.Request) any {
	v, err := s.view(ctx, r)
	if err != nil {
		return map[string]any{"id": r.ID}
	}
	redact(v, nil)
	return payload{Request: *v}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
