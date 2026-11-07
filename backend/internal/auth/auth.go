// Package auth turns a Jellyfin login into a Tipsarr session.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	CookieName = "tipsarr_session"
	// DefaultSessionTTL is the default sliding window (TIPSARR_SESSION_TTL): every use older than renewEvery
	// moves the expiry to now+TTL, so a device that is opened at least every 10 days stays signed in.
	DefaultSessionTTL = 10 * 24 * time.Hour
	// DefaultRecheckGrace is how long sessions keep working while Jellyfin cannot be asked whether their
	// account still exists (TIPSARR_SESSION_RECHECK_GRACE); after that they are refused until it answers.
	DefaultRecheckGrace = 24 * time.Hour
	renewEvery          = time.Hour
)

// Device says which client a session belongs to.
type Device struct {
	Platform   string // web, ios or android
	Name       string
	AppVersion string
}

var ErrNotConfigured = errors.New("Jellyfin is not configured yet")

type Service struct {
	store *store.Store
	// newJellyfin builds a client for a base URL; swapped in tests.
	newJellyfin func(baseURL string) *jellyfin.Client
	oidc        oidcState

	// when each user's Jellyfin account is next checked (see stillAllowed)
	mu      sync.Mutex
	nextChk map[string]time.Time
	// since when each user's re-check has kept failing (Jellyfin unreachable); in memory
	failingSince map[string]time.Time
	recheck      time.Duration
	ttl          time.Duration
	grace        time.Duration
	now          func() time.Time
}

const (
	recheckEvery = time.Hour       // a session proves its Jellyfin account still exists and is enabled this often
	recheckRetry = 5 * time.Minute // when Jellyfin cannot be asked, try again later (the session keeps working)
)

func New(s *store.Store) *Service {
	return &Service{store: s, newJellyfin: jellyfin.New, nextChk: map[string]time.Time{}, failingSince: map[string]time.Time{},
		recheck: recheckEvery, ttl: DefaultSessionTTL, grace: DefaultRecheckGrace, now: time.Now}
}

// SetSessionTTL changes the sliding session lifetime (<= 0 keeps the default).
func (s *Service) SetSessionTTL(d time.Duration) {
	if d > 0 {
		s.ttl = d
	}
}

// SessionTTL is the sliding session lifetime.
func (s *Service) SessionTTL() time.Duration { return s.ttl }

// SetRecheckGrace changes how long sessions survive a Jellyfin outage (<= 0 keeps the default).
func (s *Service) SetRecheckGrace(d time.Duration) {
	if d > 0 {
		s.grace = d
	}
}

// Login authenticates against Jellyfin and starts a session. It returns the raw
// cookie token (only its hash is stored) and the user.
func (s *Service) Login(ctx context.Context, username, password, userAgent string, dev Device) (token string, user *store.User, err error) {
	jfURL, err := s.store.GetSetting(ctx, SettingJellyfinURL)
	if err != nil {
		return "", nil, err
	}
	if jfURL == "" {
		return "", nil, ErrNotConfigured
	}
	res, err := s.newJellyfin(jfURL).Authenticate(ctx, username, password)
	if err != nil {
		return "", nil, err
	}
	// A password sign-in knows Jellyfin's answer only: it can take back a role Jellyfin gave, not one from
	// the SSO admin group (that one waits for the next SSO sign-in).
	user, err = s.store.UpsertUser(ctx, res.User.ID, res.User.Name, store.AdminGrant{
		Admin: res.User.Policy.IsAdministrator, Source: store.AdminFromJellyfin, Revokes: []string{store.AdminFromJellyfin}})
	if err != nil {
		return "", nil, err
	}

	token, err = s.startSession(ctx, user, userAgent, dev)
	return token, user, err
}

// startSession stores a new session for the user and returns the raw cookie token.
func (s *Service) startSession(ctx context.Context, user *store.User, userAgent string, dev Device) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	err := s.store.CreateSession(ctx, &store.Session{
		ID:        hash(token),
		UserID:    user.ID,
		UserAgent: userAgent,
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(s.ttl).Unix(),

		Platform: cmpOr(dev.Platform, "web"), DeviceName: dev.Name, AppVersion: dev.AppVersion, LastSeenAt: now.Unix(),
	})
	return token, err
}

// Authenticate resolves a token (cookie or Bearer) to its user and session; store.ErrNotFound if
// invalid or expired. A session used again more than renewEvery after its last use gets its
// expiry pushed forward (sess.Renewed tells the caller).
func (s *Service) Authenticate(ctx context.Context, token string) (*store.User, *store.Session, error) {
	if token == "" {
		return nil, nil, store.ErrNotFound
	}
	sess, err := s.store.LiveSession(ctx, hash(token))
	if err != nil {
		return nil, nil, err
	}
	u, err := s.store.GetUser(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if !s.stillAllowed(ctx, u) {
		return nil, nil, store.ErrNotFound
	}
	if now := time.Now(); now.Unix()-sess.LastSeenAt >= int64(renewEvery.Seconds()) {
		exp := now.Add(s.ttl).Unix()
		if err := s.store.TouchSession(ctx, sess.ID, now.Unix(), exp); err != nil {
			slog.Warn("renew session", "err", err) // the session itself is still valid
		} else {
			sess.LastSeenAt, sess.ExpiresAt, sess.Renewed = now.Unix(), exp, true
		}
	}
	return u, sess, nil
}

// SetRecheckEvery changes how often a session's Jellyfin account is checked (tests).
func (s *Service) SetRecheckEvery(d time.Duration) { s.recheck = d }

// stillAllowed asks Jellyfin (at most once per recheckEvery per user) whether the account behind a session
// still exists, is enabled and is still an administrator. A removed or disabled account loses every session;
// an admin role that came from Jellyfin is taken back when Jellyfin no longer makes the account an
// administrator (a role from the SSO admin group cannot be re-checked here and lasts until the next SSO
// sign-in; a role set by hand in Tipsarr is never touched). With no API key saved nothing can be checked and
// the session is allowed. When Jellyfin cannot be asked, sessions keep working for the grace period
// (TIPSARR_SESSION_RECHECK_GRACE), then are refused (not deleted) until Jellyfin answers again.
func (s *Service) stillAllowed(ctx context.Context, u *store.User) bool {
	now := s.now()
	s.mu.Lock()
	if now.Before(s.nextChk[u.ID]) {
		ok := s.withinGrace(u.ID, now)
		s.mu.Unlock()
		return ok
	}
	s.nextChk[u.ID] = now.Add(min(recheckRetry, s.recheck)) // also stops parallel requests from all checking
	s.mu.Unlock()

	url, _ := s.store.GetSetting(ctx, SettingJellyfinURL)
	key, _ := s.store.GetSetting(ctx, SettingJellyfinAPIKey)
	if url == "" || key == "" {
		return true
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ju, err := s.newJellyfin(url).WithToken(key).User(cctx, u.ID)
	switch {
	case jellyfin.IsNotFound(err) || (err == nil && ju.Policy.IsDisabled):
		slog.Info("Jellyfin account removed or disabled, ending its sessions", "user", u.Name)
		_ = s.store.DeleteUserSessions(context.WithoutCancel(ctx), u.ID)
		s.mu.Lock()
		delete(s.nextChk, u.ID)
		delete(s.failingSince, u.ID)
		s.mu.Unlock()
		return false
	case err == nil:
		if !ju.Policy.IsAdministrator && u.Role == store.RoleAdmin && u.AdminSource == store.AdminFromJellyfin {
			if changed, derr := s.store.DemoteJellyfinAdmin(context.WithoutCancel(ctx), u.ID); derr != nil {
				slog.Warn("demote admin", "err", derr)
			} else if changed {
				slog.Warn("no longer a Jellyfin administrator: admin role removed", "user", u.Name)
				u.Role, u.AdminSource = store.RoleUser, ""
				u.Resolve()
			}
		}
		s.mu.Lock()
		s.nextChk[u.ID] = now.Add(s.recheck)
		delete(s.failingSince, u.ID)
		s.mu.Unlock()
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.failingSince[u.ID]; !ok {
		s.failingSince[u.ID] = now
	}
	ok := s.withinGrace(u.ID, now)
	if !ok {
		slog.Warn("Jellyfin unreachable for longer than the grace period: refusing the session until it answers", "user", u.Name, "err", err)
	}
	return ok
}

// withinGrace reports whether a failing re-check is still inside the grace period (callers hold s.mu).
func (s *Service) withinGrace(userID string, now time.Time) bool {
	since, failing := s.failingSince[userID]
	return !failing || now.Sub(since) <= s.grace
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hash(token))
}

func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func cmpOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
