// Package auth turns a Jellyfin login into a Tipsarr session.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

const (
	CookieName = "tipsarr_session"
	SessionTTL = 30 * 24 * time.Hour
)

var ErrNotConfigured = errors.New("Jellyfin is not configured yet")

type Service struct {
	store *store.Store
	// newJellyfin builds a client for a base URL; swapped in tests.
	newJellyfin func(baseURL string) *jellyfin.Client
}

func New(s *store.Store) *Service {
	return &Service{store: s, newJellyfin: jellyfin.New}
}

// Login authenticates against Jellyfin and starts a session. It returns the raw
// cookie token (only its hash is stored) and the user.
func (s *Service) Login(ctx context.Context, username, password, userAgent string) (token string, user *store.User, err error) {
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

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	now := time.Now()
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	err = s.store.CreateSession(ctx, &store.Session{
		ID:        hash(token),
		UserID:    user.ID,
		UserAgent: userAgent,
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(SessionTTL).Unix(),
	})
	return token, user, err
}

// Authenticate resolves a cookie token to its user; store.ErrNotFound if invalid or expired.
func (s *Service) Authenticate(ctx context.Context, token string) (*store.User, error) {
	if token == "" {
		return nil, store.ErrNotFound
	}
	return s.store.SessionUser(ctx, hash(token))
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, hash(token))
}

func hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
