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
	// SessionTTL is a sliding window: every use older than renewEvery moves the expiry to now+SessionTTL,
	// so a device that is opened at least every 10 days stays signed in.
	SessionTTL = 10 * 24 * time.Hour
	renewEvery = time.Hour
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
	recheck time.Duration
}

const (
	recheckEvery = time.Hour       // a session proves its Jellyfin account still exists and is enabled this often
	recheckRetry = 5 * time.Minute // when Jellyfin cannot be asked, try again later (the session keeps working)
)

func New(s *store.Store) *Service {
	return &Service{store: s, newJellyfin: jellyfin.New, nextChk: map[string]time.Time{}, recheck: recheckEvery}
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
	user, err = s.store.UpsertUser(ctx, res.User.ID, res.User.Name, res.User.Policy.IsAdministrator)
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
		ExpiresAt: now.Add(SessionTTL).Unix(),

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
		exp := now.Add(SessionTTL).Unix()
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

// stillAllowed asks Jellyfin (at most once per recheckEvery per user) whether the account behind
// a session still exists and is enabled; if not, every session of that user is ended. It fails
// open: no API key saved, or Jellyfin unreachable, never locks anyone out.
func (s *Service) stillAllowed(ctx context.Context, u *store.User) bool {
	s.mu.Lock()
	if time.Now().Before(s.nextChk[u.ID]) {
		s.mu.Unlock()
		return true
	}
	s.nextChk[u.ID] = time.Now().Add(min(recheckRetry, s.recheck)) // also stops parallel requests from all checking
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
		s.mu.Unlock()
		return false
	case err == nil:
		s.mu.Lock()
		s.nextChk[u.ID] = time.Now().Add(s.recheck)
		s.mu.Unlock()
	}
	return true
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
