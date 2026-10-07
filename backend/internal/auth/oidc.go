package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/clients/jellyfin"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Reasons a single sign-on attempt is refused; the login page shows the classic form again.
var (
	ErrSSONoUser    = errors.New("no Jellyfin account matches this single sign-on user")
	ErrSSODisabled  = errors.New("this Jellyfin account is disabled")
	ErrSSOUnusable  = errors.New("single sign-on is not set up (Jellyfin API key or provider missing)")
	ErrSSOProvider  = errors.New("the single sign-on provider did not accept the request")
	ErrSSOState     = errors.New("the sign-in attempt expired or does not belong to this browser")
	ErrSSOUnmatched = errors.New("the provider did not send a username")
)

const pendingTTL = 10 * time.Minute

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	AdminGroup   string
	GroupsClaim  string
}

// Enabled: single sign-on is offered once the provider, client and secret are all saved.
func (c OIDCConfig) Enabled() bool { return c.Issuer != "" && c.ClientID != "" && c.ClientSecret != "" }

func (s *Service) OIDCConfig(ctx context.Context) (OIDCConfig, error) {
	var c OIDCConfig
	for key, dst := range map[string]*string{
		SettingOIDCIssuer: &c.Issuer, SettingOIDCClientID: &c.ClientID, SettingOIDCClientSecret: &c.ClientSecret,
		SettingOIDCAdminGroup: &c.AdminGroup, SettingOIDCGroupsClaim: &c.GroupsClaim,
	} {
		v, err := s.store.GetSetting(ctx, key)
		if err != nil {
			return c, err
		}
		*dst = strings.TrimSpace(v)
	}
	c.Issuer = strings.TrimRight(c.Issuer, "/")
	if c.GroupsClaim == "" {
		c.GroupsClaim = "groups"
	}
	return c, nil
}

type pending struct {
	nonce, verifier string
	expires         time.Time
}

type oidcState struct {
	mu       sync.Mutex
	pending  map[string]pending // state -> what the callback needs
	provider *oidc.Provider
	issuer   string
	fetched  time.Time
}

func (s *Service) discover(ctx context.Context, issuer string) (*oidc.Provider, error) {
	s.oidc.mu.Lock()
	defer s.oidc.mu.Unlock()
	if s.oidc.provider != nil && s.oidc.issuer == issuer && time.Since(s.oidc.fetched) < time.Hour {
		return s.oidc.provider, nil
	}
	p, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSSOProvider, err)
	}
	s.oidc.provider, s.oidc.issuer, s.oidc.fetched = p, issuer, time.Now()
	return p, nil
}

// CheckOIDC verifies the provider's discovery document (used when saving the settings).
func (s *Service) CheckOIDC(ctx context.Context, issuer string) error {
	s.oidc.mu.Lock()
	s.oidc.provider = nil
	s.oidc.mu.Unlock()
	_, err := s.discover(ctx, strings.TrimRight(issuer, "/"))
	return err
}

func randomString(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) oauthConfig(c OIDCConfig, p *oidc.Provider, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID: c.ClientID, ClientSecret: c.ClientSecret, Endpoint: p.Endpoint(), RedirectURL: redirectURL,
		Scopes: []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
}

// BeginOIDC returns the provider URL to send the browser to, and the state that must come back
// in the callback (also kept in a cookie so the browser that started the attempt finishes it).
func (s *Service) BeginOIDC(ctx context.Context, redirectURL string) (authURL, state string, err error) {
	c, err := s.OIDCConfig(ctx)
	if err != nil {
		return "", "", err
	}
	if !c.Enabled() {
		return "", "", ErrSSOUnusable
	}
	p, err := s.discover(ctx, c.Issuer)
	if err != nil {
		return "", "", err
	}
	state, nonce, verifier := randomString(24), randomString(24), oauth2.GenerateVerifier()
	s.oidc.mu.Lock()
	if s.oidc.pending == nil {
		s.oidc.pending = map[string]pending{}
	}
	for k, v := range s.oidc.pending { // forget abandoned attempts
		if time.Now().After(v.expires) {
			delete(s.oidc.pending, k)
		}
	}
	if len(s.oidc.pending) > 500 {
		s.oidc.mu.Unlock()
		return "", "", ErrSSOState
	}
	s.oidc.pending[state] = pending{nonce: nonce, verifier: verifier, expires: time.Now().Add(pendingTTL)}
	s.oidc.mu.Unlock()
	url := s.oauthConfig(c, p, redirectURL).AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oidc.Nonce(nonce))
	return url, state, nil
}

// CompleteOIDC finishes the sign-in: it exchanges the code, checks the ID token, finds the
// matching Jellyfin account by username and starts a session. Every failure is one of the
// ErrSSO* values so the caller can send the person back to the classic form.
func (s *Service) CompleteOIDC(ctx context.Context, redirectURL, state, code, userAgent string) (string, *store.User, error) {
	s.oidc.mu.Lock()
	pend, ok := s.oidc.pending[state]
	delete(s.oidc.pending, state)
	s.oidc.mu.Unlock()
	if !ok || time.Now().After(pend.expires) {
		return "", nil, ErrSSOState
	}
	c, err := s.OIDCConfig(ctx)
	if err != nil {
		return "", nil, err
	}
	if !c.Enabled() {
		return "", nil, ErrSSOUnusable
	}
	p, err := s.discover(ctx, c.Issuer)
	if err != nil {
		return "", nil, err
	}
	tok, err := s.oauthConfig(c, p, redirectURL).Exchange(ctx, code, oauth2.VerifierOption(pend.verifier))
	if err != nil {
		slog.Warn("sso: code exchange failed", "err", err)
		return "", nil, ErrSSOProvider
	}
	rawID, _ := tok.Extra("id_token").(string)
	idt, err := p.Verifier(&oidc.Config{ClientID: c.ClientID}).Verify(ctx, rawID)
	if err != nil || idt.Nonce != pend.nonce {
		slog.Warn("sso: ID token rejected", "err", err)
		return "", nil, ErrSSOProvider
	}
	var claims map[string]any
	if err := idt.Claims(&claims); err != nil {
		return "", nil, ErrSSOProvider
	}
	username := firstString(claims, "preferred_username", "name", "nickname")
	if username == "" {
		return "", nil, ErrSSOUnmatched
	}

	jfURL, err := s.store.GetSetting(ctx, SettingJellyfinURL)
	if err != nil {
		return "", nil, err
	}
	key, err := s.store.GetSetting(ctx, SettingJellyfinAPIKey)
	if err != nil {
		return "", nil, err
	}
	if jfURL == "" || key == "" {
		return "", nil, ErrSSOUnusable
	}
	users, err := s.newJellyfin(jfURL).WithToken(key).Users(ctx)
	if err != nil {
		slog.Warn("sso: cannot list Jellyfin users", "err", err)
		return "", nil, ErrSSOUnusable
	}
	var match *jellyfin.User
	for i := range users {
		if strings.EqualFold(users[i].Name, username) {
			match = &users[i]
			break
		}
	}
	if match == nil {
		slog.Info("sso: no Jellyfin user for this identity", "username", username)
		return "", nil, ErrSSONoUser
	}
	if match.Policy.IsDisabled {
		return "", nil, ErrSSODisabled
	}
	admin := match.Policy.IsAdministrator || (c.AdminGroup != "" && inGroups(claims[c.GroupsClaim], c.AdminGroup))
	user, err := s.store.UpsertUser(ctx, strings.ToLower(strings.ReplaceAll(match.ID, "-", "")), match.Name, admin)
	if err != nil {
		return "", nil, err
	}
	token, err := s.startSession(ctx, user, userAgent)
	return token, user, err
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// inGroups reports whether a groups claim (a list, or one string) contains the group.
func inGroups(claim any, group string) bool {
	switch v := claim.(type) {
	case []any:
		for _, g := range v {
			if s, ok := g.(string); ok && strings.EqualFold(s, group) {
				return true
			}
		}
	case string:
		return strings.EqualFold(v, group)
	}
	return false
}
