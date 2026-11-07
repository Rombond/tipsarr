// Package demo holds what the public demo needs: the DEMO mode switches of the Tipsarr server (refused actions,
// outbound network lock) and the fake services and TMDB proxy run by cmd/demo-gateway.
//
// In demo mode (TIPSARR_DEMO_MODE=true) a Tipsarr process is a throwaway sandbox: anyone may be its admin, so it must
// not be able to reach anything but the gateway. Three layers make sure of that:
//   - Restrict: every outbound connection (HTTP and LDAP) may only go to the addresses in TIPSARR_DEMO_ALLOW_DIAL;
//   - Guard: the API calls whose only purpose is to reach the outside (webhook test, push relay, LDAP, OIDC ...) answer
//     403 with the code demo_disabled;
//   - main turns dry-run on for good, keeps webhooks offline and skips the heavy background jobs.
package demo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// ErrorCode is the API error code of every action refused in demo mode.
const ErrorCode = "demo_disabled"

// Accounts of the seeded demo (the fake Jellyfin accepts Password for each).
const (
	AdminUser = "alice"
	PlainUser = "bob"
	Password  = "demo"
)

// Config is the demo part of the process configuration.
type Config struct {
	Enabled bool
	// AllowDial lists the only "ip:port" addresses this process may connect to (the gateway's internal listener).
	AllowDial []string
	// PublicURL is the locked public address of the demo (shown in links), from TIPSARR_DEMO_PUBLIC_URL.
	PublicURL string
}

// FromEnv reads TIPSARR_DEMO_MODE, TIPSARR_DEMO_ALLOW_DIAL and TIPSARR_DEMO_PUBLIC_URL.
func FromEnv() Config {
	v := strings.ToLower(os.Getenv("TIPSARR_DEMO_MODE"))
	c := Config{Enabled: v == "true" || v == "1", PublicURL: strings.TrimRight(os.Getenv("TIPSARR_DEMO_PUBLIC_URL"), "/")}
	for _, a := range strings.Split(os.Getenv("TIPSARR_DEMO_ALLOW_DIAL"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			c.AllowDial = append(c.AllowDial, a)
		}
	}
	return c
}

// ErrDialRefused is returned for any connection outside the allow list.
var ErrDialRefused = errors.New("demo: outbound connection refused")

// DialControl returns a net.Dialer Control func that only lets connections to the allowed "ip:port" through. It runs
// on the resolved address, so a host name that resolves elsewhere is refused too.
func DialControl(allow []string) func(network, address string, _ syscall.RawConn) error {
	ok := map[string]bool{}
	for _, a := range allow {
		host, port, err := net.SplitHostPort(a)
		if err != nil {
			continue
		}
		if host == "localhost" {
			host = "127.0.0.1"
		}
		if ip := net.ParseIP(host); ip != nil {
			ok[net.JoinHostPort(ip.String(), port)] = true
		}
	}
	return func(_, address string, _ syscall.RawConn) error {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return ErrDialRefused
		}
		if ip := net.ParseIP(host); ip != nil && ok[net.JoinHostPort(ip.String(), port)] {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrDialRefused, address)
	}
}

// Restrict replaces http.DefaultTransport (which every client of the server uses) with one that can only reach the
// allowed addresses, and ignores any proxy settings from the environment.
func Restrict(allow []string) {
	d := &net.Dialer{Timeout: 5 * time.Second, Control: DialControl(allow)}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = d.DialContext
	http.DefaultTransport = t
}

// blocked are the API calls refused in demo mode: they exist only to reach another service.
var blocked = []struct {
	method string
	path   *regexp.Regexp
}{
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/webhooks/[^/]+/test$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/push/register$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/ldap/import-jellyfin$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/teasers/files/test$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/sync/(teaser-pool|ytdlp-update|trailer-files-sync)$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/admin/users/import$`)},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/me/avatar$`)},
	{http.MethodPut, regexp.MustCompile(`^/api/v1/me/devices/current$`)},
	{http.MethodGet, regexp.MustCompile(`^/api/v1/auth/oidc/`)},
}

// Blocked reports whether the call is refused in demo mode.
func Blocked(method, path string) bool {
	for _, b := range blocked {
		if b.method == method && b.path.MatchString(path) {
			return true
		}
	}
	return false
}

// Guard answers the blocked calls with 403 demo_disabled (the same error shape as the API's other errors).
func Guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Blocked(r.Method, r.URL.Path) {
			WriteRefusal(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RefusalDetail is the English text of a refusal.
const RefusalDetail = "disabled in the demo: this action would reach another service"

// WriteRefusal writes the 403 demo_disabled answer.
func WriteRefusal(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title": "Forbidden", "status": http.StatusForbidden, "detail": RefusalDetail,
		"errors": []map[string]any{{"message": RefusalDetail, "location": "code", "value": ErrorCode}},
	})
}
