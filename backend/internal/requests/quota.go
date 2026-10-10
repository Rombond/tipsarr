package requests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/perms"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// ErrQuota means the person has used up their request limit for now.
var ErrQuota = errors.New("request limit reached")

// QuotaError says which limit was hit and when a request becomes possible again.
type QuotaError struct {
	Type     string // movie | tv
	Window   QuotaWindow
	Needed   int
	ResetsAt int64
}

func (e *QuotaError) Error() string {
	what := "requests"
	if e.Type == "tv" {
		what = "seasons"
	}
	msg := fmt.Sprintf("request limit reached: %d %s every %d days", e.Window.Limit, what, e.Window.Days)
	if e.ResetsAt > 0 {
		msg += ", next possible " + time.Unix(e.ResetsAt, 0).UTC().Format("2006-01-02")
	}
	return msg
}

func (e *QuotaError) Unwrap() error { return ErrQuota }

// QuotaWindow is one limit: Limit requests (movies) or seasons (shows) every Days days. Limit 0 = no limit.
type QuotaWindow struct {
	Days       int   `json:"days"`
	Limit      int   `json:"limit"`
	Used       int   `json:"used"`
	Remaining  *int  `json:"remaining,omitempty" doc:"Absent when there is no limit"`
	Restricted bool  `json:"restricted" doc:"The limit is reached: new requests are refused until ResetsAt"`
	ResetsAt   int64 `json:"resetsAt,omitempty" doc:"Unix time when the oldest counted request leaves the window"`
}

// Quota is a person's two limits.
type Quota struct {
	Movie QuotaWindow `json:"movie"`
	TV    QuotaWindow `json:"tv"`
}

func (s *Service) window(ctx context.Context, u *store.User, mediaType string) (QuotaWindow, error) {
	var limit, days int
	if mediaType == "movie" {
		limit = s.store.SettingInt(ctx, store.SettingQuotaMovieLimit, 0)
		days = s.store.SettingInt(ctx, store.SettingQuotaMovieDays, 0)
		if u.MovieQuotaLimit != nil {
			limit = *u.MovieQuotaLimit
		}
		if u.MovieQuotaDays != nil {
			days = *u.MovieQuotaDays
		}
	} else {
		limit = s.store.SettingInt(ctx, store.SettingQuotaTVLimit, 0)
		days = s.store.SettingInt(ctx, store.SettingQuotaTVDays, 0)
		if u.TVQuotaLimit != nil {
			limit = *u.TVQuotaLimit
		}
		if u.TVQuotaDays != nil {
			days = *u.TVQuotaDays
		}
	}
	w := QuotaWindow{Days: days, Limit: limit}
	if u.Can(perms.ManageRequests) || limit <= 0 || days <= 0 {
		w.Limit = 0 // managers and admins have no limit; neither has anyone without a limit and a period
		return w, nil
	}
	now := s.now()
	used, oldest, err := s.store.QuotaUsage(ctx, u.ID, mediaType, now.AddDate(0, 0, -days).Unix())
	if err != nil {
		return w, err
	}
	w.Used = used
	rem := max(limit-used, 0)
	w.Remaining = &rem
	w.Restricted = rem == 0
	if w.Restricted && oldest > 0 {
		w.ResetsAt = oldest + int64(days)*86400
	}
	return w, nil
}

// Quota returns what the person may still request.
func (s *Service) Quota(ctx context.Context, u *store.User) (*Quota, error) {
	m, err := s.window(ctx, u, "movie")
	if err != nil {
		return nil, err
	}
	t, err := s.window(ctx, u, "tv")
	if err != nil {
		return nil, err
	}
	return &Quota{Movie: m, TV: t}, nil
}

// checkQuota refuses a request that needs more than what is left (one for a movie, one per season for a show).
func (s *Service) checkQuota(ctx context.Context, u *store.User, mediaType string, needed int) error {
	w, err := s.window(ctx, u, mediaType)
	if err != nil {
		return err
	}
	if w.Limit > 0 && w.Used+needed > w.Limit {
		reset := w.ResetsAt
		if reset == 0 {
			reset = s.now().Unix() + int64(w.Days)*86400
		}
		return &QuotaError{Type: mediaType, Window: w, Needed: needed, ResetsAt: reset}
	}
	return nil
}
