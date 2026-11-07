package push

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	errUnauthorized = errors.New("the relay refused the key (HTTP 401)")
	versionRe       = regexp.MustCompile(`^[A-Za-z0-9._+-]{1,32}$`)
)

// quotaError: the relay's daily quota for this key is spent.
type quotaError struct{ retryAfter int }

func (e *quotaError) Error() string {
	return fmt.Sprintf("daily quota reached, retry in %ds", e.retryAfter)
}

// RegisterError is a refusal of POST /v1/register. Code is the relay's error code
// (invalid_request, unknown_app, invalid_install_id, invalid_server_url, invalid_version,
// registration_not_allowed, registration_closed, rate_limited, registration_cap) or
// "unreachable" when the relay could not be reached or answered something unexpected.
type RegisterError struct {
	Status     int
	Code       string
	RetryAfter int // seconds, for 429
}

func (e *RegisterError) Error() string {
	if e.Status == 0 {
		return "the push relay could not be reached"
	}
	return fmt.Sprintf("the push relay refused the registration: %s (HTTP %d)", e.Code, e.Status)
}

// Registration is the relay's answer to a successful registration.
type Registration struct {
	KeyID      int64 `json:"keyId"`
	Rotated    bool  `json:"rotated"`
	DailyQuota int   `json:"dailyQuota"`
}

// InstallID returns this server's install id, generating and storing it on first use.
func (s *Service) InstallID(ctx context.Context) (string, error) {
	if v, err := s.store.GetSetting(ctx, SettingInstallID); err != nil || v != "" {
		return v, err
	}
	b := make([]byte, 32) // 256 bits, 43 characters
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(b)
	return id, s.store.SetSetting(ctx, SettingInstallID, id)
}

// Register asks the relay at relayURL for a public key (open registration) and saves it with the address,
// marked public, and switches push on. Registering again rotates the key; the old one stops working.
func (s *Service) Register(ctx context.Context, relayURL string) (*Registration, error) {
	relayURL = strings.TrimRight(strings.TrimSpace(relayURL), "/")
	if err := ValidURL(relayURL); err != nil {
		return nil, err
	}
	reg, key, err := s.register(ctx, relayURL)
	if err != nil {
		return nil, err
	}
	for k, v := range map[string]string{
		SettingURL: relayURL, SettingKey: key, SettingKeyPublic: "true", SettingDailyQuota: strconv.Itoa(reg.DailyQuota),
		SettingPausedUntil: "", SettingEnabled: "true",
	} {
		if err := s.store.SetSetting(ctx, k, v); err != nil {
			return nil, err
		}
	}
	s.regMu.Lock()
	s.regFailed = time.Time{}
	s.regMu.Unlock()
	return reg, nil
}

func (s *Service) register(ctx context.Context, relayURL string) (*Registration, string, error) {
	id, err := s.InstallID(ctx)
	if err != nil {
		return nil, "", err
	}
	body := map[string]string{"app": AppName, "installId": id}
	if s.ServerURL != nil {
		if u := s.ServerURL(ctx); u != "" && ValidURL(u) == nil && len(u) <= 256 {
			body["serverUrl"] = u
		}
	}
	if versionRe.MatchString(s.Version) {
		body["version"] = s.Version
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, relayURL+"/v1/register", bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Tipsarr")
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, "", &RegisterError{}
	}
	defer resp.Body.Close()
	var out struct {
		Registration
		Key   string `json:"key"`
		App   string `json:"app"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out)
	switch {
	case (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) && out.Key != "":
		return &out.Registration, out.Key, nil
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		code := out.Error
		if code == "" {
			code = "invalid_request"
		}
		return nil, "", &RegisterError{Status: resp.StatusCode, Code: code, RetryAfter: retryAfter(resp)}
	}
	return nil, "", &RegisterError{Status: resp.StatusCode, Code: "unreachable"}
}

// reregister replaces a public key the relay refused (401), once, with the same installId.
// After a failure it waits regBackoff before trying again, so a refusing relay is not hammered.
func (s *Service) reregister(ctx context.Context, relayURL, refused string) (string, error) {
	s.regMu.Lock()
	defer s.regMu.Unlock()
	// another send may have registered while this one waited: use its key rather than rotating again
	if k, _ := s.store.GetSetting(ctx, SettingKey); k != "" && k != refused {
		return k, nil
	}
	if !s.regFailed.IsZero() && time.Since(s.regFailed) < s.regBackoff {
		return "", errors.New("re-registration failed recently, waiting")
	}
	reg, key, err := s.register(ctx, relayURL)
	if err != nil {
		s.regFailed = time.Now()
		return "", err
	}
	s.regFailed = time.Time{}
	if err := s.store.SetSetting(ctx, SettingKey, key); err != nil {
		return "", err
	}
	_ = s.store.SetSetting(ctx, SettingDailyQuota, strconv.Itoa(reg.DailyQuota))
	slog.Info("push: registered again with the relay", "keyId", reg.KeyID)
	return key, nil
}

// pause stops sending for retryAfter seconds (daily quota spent).
func (s *Service) pause(ctx context.Context, retryAfter int) {
	until := time.Now().Add(time.Duration(retryAfter) * time.Second).Unix()
	slog.Warn("push: daily quota of the relay reached, sending paused", "until", time.Unix(until, 0).UTC())
	if err := s.store.SetSetting(ctx, SettingPausedUntil, strconv.FormatInt(until, 10)); err != nil {
		slog.Warn("push: save pause", "err", err)
	}
}

// PausedUntil returns when sending resumes after the relay's daily quota was reached, 0 when not paused.
func (s *Service) PausedUntil(ctx context.Context) int64 {
	v, _ := s.store.GetSetting(ctx, SettingPausedUntil)
	n, _ := strconv.ParseInt(v, 10, 64)
	if n <= time.Now().Unix() {
		return 0
	}
	return n
}

func retryAfter(resp *http.Response) int {
	n, _ := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After")))
	return max(n, 0)
}
