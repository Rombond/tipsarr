// Package issues lets users report a problem with a title (bad video, missing subtitles...)
// and lets admins follow it up in a comment thread.
package issues

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/Rombond/tipsarr/backend/internal/events"
	"github.com/Rombond/tipsarr/backend/internal/media"
	"github.com/Rombond/tipsarr/backend/internal/notify"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

var (
	ErrInvalid   = errors.New("invalid issue")
	ErrForbidden = errors.New("not allowed")
)

var Kinds = []string{"video", "audio", "subtitles", "other"}

type Service struct {
	store  *store.Store
	media  *media.Service
	hub    *events.Hub
	notify *notify.Service
}

func New(s *store.Store, m *media.Service, h *events.Hub, n *notify.Service) *Service {
	return &Service{store: s, media: m, hub: h, notify: n}
}

type IssueUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueView struct {
	ID           string     `json:"id"`
	Type         string     `json:"type" enum:"movie,tv"`
	TMDBID       int        `json:"tmdbId"`
	Title        string     `json:"title"`
	PosterPath   string     `json:"posterPath,omitempty"`
	Kind         string     `json:"kind" enum:"video,audio,subtitles,other"`
	Season       int        `json:"season,omitempty"`
	Episode      int        `json:"episode,omitempty"`
	Status       string     `json:"status" enum:"open,resolved"`
	CreatedBy    IssueUser  `json:"createdBy"`
	ResolvedBy   *IssueUser `json:"resolvedBy,omitempty"`
	CommentCount int        `json:"commentCount"`
	CreatedAt    int64      `json:"createdAt"`
	UpdatedAt    int64      `json:"updatedAt"`
}

type IssueComment struct {
	ID        string    `json:"id"`
	User      IssueUser `json:"user"`
	Message   string    `json:"message"`
	CreatedAt int64     `json:"createdAt"`
}

type Thread struct {
	IssueView
	Comments []IssueComment `json:"comments"`
}

func canSee(u *store.User, i *store.Issue) bool {
	return u.Role == store.RoleAdmin || i.CreatedBy == u.ID
}

type CreateParams struct {
	Type    string
	TMDBID  int
	Kind    string
	Season  int
	Episode int
	Message string
}

func (s *Service) Create(ctx context.Context, u *store.User, p CreateParams) (*Thread, error) {
	msg := strings.TrimSpace(p.Message)
	valid := false
	for _, k := range Kinds {
		valid = valid || k == p.Kind
	}
	if !valid || (p.Type != "movie" && p.Type != "tv") || p.TMDBID <= 0 || msg == "" || utf8.RuneCountInString(msg) > 2000 {
		return nil, ErrInvalid
	}
	if p.Type == "movie" {
		p.Season, p.Episode = 0, 0
	}
	d, err := s.media.Detail(ctx, media.Opts{Language: u.Language, Region: u.Region}, p.Type, p.TMDBID)
	if err != nil {
		return nil, err
	}
	i := &store.Issue{
		ID: store.NewID(), MediaType: p.Type, TMDBID: int64(p.TMDBID), Title: d.Title, PosterPath: d.PosterPath,
		Kind: p.Kind, SeasonNumber: p.Season, EpisodeNumber: p.Episode, Status: store.IssueOpen, CreatedBy: u.ID,
	}
	if err := s.store.CreateIssue(ctx, i, msg); err != nil {
		return nil, err
	}
	v, err := s.detail(ctx, i)
	if err != nil {
		return nil, err
	}
	s.changed(i)
	s.notify.Dispatch(notify.IssueCreated, map[string]any{"issue": v.IssueView, "message": msg})
	return v, nil
}

func (s *Service) views(ctx context.Context, rows []store.Issue) ([]IssueView, map[string][]store.IssueComment, error) {
	ids := make([]string, len(rows))
	people := map[string]bool{}
	for i, r := range rows {
		ids[i] = r.ID
		people[r.CreatedBy] = true
		if r.ResolvedBy != "" {
			people[r.ResolvedBy] = true
		}
	}
	comments, err := s.store.IssueComments(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	byIssue := map[string][]store.IssueComment{}
	for _, c := range comments {
		byIssue[c.IssueID] = append(byIssue[c.IssueID], c)
		people[c.UserID] = true
	}
	pl := make([]string, 0, len(people))
	for id := range people {
		pl = append(pl, id)
	}
	names, err := s.store.UserNames(ctx, pl)
	if err != nil {
		return nil, nil, err
	}
	out := make([]IssueView, len(rows))
	for i, r := range rows {
		v := IssueView{
			ID: r.ID, Type: r.MediaType, TMDBID: int(r.TMDBID), Title: r.Title, PosterPath: r.PosterPath, Kind: r.Kind,
			Season: r.SeasonNumber, Episode: r.EpisodeNumber, Status: r.Status,
			CreatedBy: IssueUser{ID: r.CreatedBy, Name: names[r.CreatedBy]}, CommentCount: len(byIssue[r.ID]),
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
		if r.ResolvedBy != "" {
			v.ResolvedBy = &IssueUser{ID: r.ResolvedBy, Name: names[r.ResolvedBy]}
		}
		out[i] = v
	}
	return out, byIssue, nil
}

func (s *Service) detail(ctx context.Context, i *store.Issue) (*Thread, error) {
	vs, comments, err := s.views(ctx, []store.Issue{*i})
	if err != nil {
		return nil, err
	}
	names, err := s.store.UserNames(ctx, commentUsers(comments[i.ID]))
	if err != nil {
		return nil, err
	}
	d := &Thread{IssueView: vs[0], Comments: []IssueComment{}}
	for _, c := range comments[i.ID] {
		d.Comments = append(d.Comments, IssueComment{ID: c.ID, User: IssueUser{ID: c.UserID, Name: names[c.UserID]}, Message: c.Message, CreatedAt: c.CreatedAt})
	}
	return d, nil
}

func commentUsers(cs []store.IssueComment) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cs {
		if !seen[c.UserID] {
			seen[c.UserID] = true
			out = append(out, c.UserID)
		}
	}
	return out
}

type ListParams struct {
	Filter     string // all | open | resolved
	MediaType  string // with TMDBID: issues of one title
	TMDBID     int
	Take, Skip int
}

type ListResult struct {
	Total int
	Items []IssueView
}

func (s *Service) List(ctx context.Context, u *store.User, p ListParams) (*ListResult, error) {
	f := store.IssueFilter{Take: p.Take, Skip: p.Skip, MediaType: p.MediaType, TMDBID: p.TMDBID}
	if u.Role != store.RoleAdmin {
		f.CreatedBy = u.ID
	}
	if p.Filter == "open" || p.Filter == "resolved" {
		f.Status = p.Filter
	}
	rows, total, err := s.store.ListIssues(ctx, f)
	if err != nil {
		return nil, err
	}
	items, _, err := s.views(ctx, rows)
	return &ListResult{Total: total, Items: items}, err
}

func (s *Service) Counts(ctx context.Context, u *store.User) (open, resolved int, err error) {
	by := u.ID
	if u.Role == store.RoleAdmin {
		by = ""
	}
	return s.store.IssueCounts(ctx, by)
}

func (s *Service) Get(ctx context.Context, u *store.User, id string) (*Thread, error) {
	i, err := s.store.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canSee(u, i) {
		return nil, store.ErrNotFound
	}
	return s.detail(ctx, i)
}

func (s *Service) Comment(ctx context.Context, u *store.User, id, message string) (*Thread, error) {
	msg := strings.TrimSpace(message)
	if msg == "" || utf8.RuneCountInString(msg) > 2000 {
		return nil, ErrInvalid
	}
	i, err := s.store.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canSee(u, i) {
		return nil, store.ErrNotFound
	}
	if err := s.store.AddIssueComment(ctx, &store.IssueComment{ID: store.NewID(), IssueID: id, UserID: u.ID, Message: msg}); err != nil {
		return nil, err
	}
	d, err := s.detail(ctx, i)
	if err != nil {
		return nil, err
	}
	s.changed(i)
	s.notify.Dispatch(notify.IssueCommented, map[string]any{"issue": d.IssueView, "comment": msg, "by": IssueUser{ID: u.ID, Name: u.Name}})
	return d, nil
}

// SetResolved resolves or reopens an issue. Admins and the reporter may do either.
func (s *Service) SetResolved(ctx context.Context, u *store.User, id string, resolved bool) (*Thread, error) {
	i, err := s.store.GetIssue(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canSee(u, i) {
		return nil, store.ErrNotFound
	}
	if resolved {
		i.Status, i.ResolvedBy = store.IssueResolved, u.ID
	} else {
		i.Status, i.ResolvedBy = store.IssueOpen, ""
	}
	if err := s.store.UpdateIssue(ctx, i); err != nil {
		return nil, err
	}
	d, err := s.detail(ctx, i)
	if err != nil {
		return nil, err
	}
	s.changed(i)
	if resolved {
		s.notify.Dispatch(notify.IssueResolved, map[string]any{"issue": d.IssueView})
	}
	return d, nil
}

func (s *Service) Delete(ctx context.Context, u *store.User, id string) error {
	if u.Role != store.RoleAdmin {
		return ErrForbidden
	}
	i, err := s.store.GetIssue(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.DeleteIssue(ctx, id); err != nil {
		return err
	}
	s.hub.Publish("issue.updated", i.CreatedBy, map[string]any{"id": id, "status": "deleted"}, false)
	return nil
}

func (s *Service) changed(i *store.Issue) {
	s.hub.Publish("issue.updated", i.CreatedBy, map[string]any{"id": i.ID, "status": i.Status}, false)
}
