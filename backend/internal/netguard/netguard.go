// Package netguard is the egress policy of outbound requests whose address an admin types in
// (webhooks, the Radarr/Sonarr probe). The check runs in the dialer, on the address actually
// connected to, after DNS resolution: a name that resolves to a forbidden address (or is rebound
// to one between two lookups) is refused like the literal address.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// ErrRefused is returned (wrapped) when the policy refuses an address.
var ErrRefused = errors.New("address refused by the outbound policy")

// cgnat is 100.64.0.0/10 (carrier-grade NAT, often a VPN such as Tailscale).
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// AlwaysBlocked reports addresses no outbound request may reach, whatever the settings:
// loopback, link-local (cloud metadata 169.254.169.254 included), unspecified and multicast.
func AlwaysBlocked(ip netip.Addr) bool {
	ip = ip.Unmap()
	if testLoopback.Load() && ip.IsLoopback() {
		return false
	}
	return !ip.IsValid() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

var testLoopback atomic.Bool

// AllowLoopbackForTests lets loopback through every policy of the process (httptest servers).
// Never call it outside tests.
func AllowLoopbackForTests() { testLoopback.Store(true) }

// Private reports private-network addresses: RFC 1918, unique local IPv6 (fc00::/7) and CGNAT.
func Private(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsPrivate() || cgnat.Contains(ip)
}

// Policy decides which addresses a dialer may connect to.
type Policy struct {
	// AllowPrivate lets private-network addresses through (the always-blocked ones never pass).
	AllowPrivate bool
	// Allowed returns extra private targets: "host", "host:port" or a CIDR. Called on every dial.
	AllowList func() []string
}

// allows reports whether host:port (as dialed, before resolution) resolved to ip may be reached.
func (p Policy) allows(host, port string, ip netip.Addr) bool {
	ip = ip.Unmap()
	if AlwaysBlocked(ip) {
		return false
	}
	if !Private(ip) || p.AllowPrivate {
		return true
	}
	if p.AllowList == nil {
		return false
	}
	return MatchAllowList(p.AllowList(), host, port, ip)
}

// MatchAllowList reports whether host (name or literal), port or the resolved ip is in the list.
func MatchAllowList(list []string, host, port string, ip netip.Addr) bool {
	host = strings.ToLower(strings.Trim(host, "[]"))
	for _, raw := range list {
		e := strings.ToLower(strings.TrimSpace(raw))
		if e == "" {
			continue
		}
		if pf, err := netip.ParsePrefix(e); err == nil {
			if pf.Contains(ip.Unmap()) {
				return true
			}
			continue
		}
		if a, err := netip.ParseAddr(strings.Trim(e, "[]")); err == nil {
			if a.Unmap() == ip.Unmap() {
				return true
			}
			continue
		}
		if h, pt, err := net.SplitHostPort(e); err == nil {
			h = strings.Trim(h, "[]")
			if pt == port && (h == host || addrEq(h, ip)) {
				return true
			}
			continue
		}
		if e == host {
			return true
		}
	}
	return false
}

func addrEq(s string, ip netip.Addr) bool {
	a, err := netip.ParseAddr(s)
	return err == nil && a.Unmap() == ip.Unmap()
}

// ValidAllowEntry reports whether s is a usable allow-list entry (host, host:port or CIDR).
func ValidAllowEntry(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \\@?#") {
		return false
	}
	if strings.Contains(s, "/") {
		_, err := netip.ParsePrefix(s)
		return err == nil
	}
	host := s
	if h, p, err := net.SplitHostPort(s); err == nil {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return false
		}
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.Contains(host, ":") {
		_, err := netip.ParseAddr(host)
		return err == nil
	}
	return host != ""
}

// Resolver looks names up; tests replace it.
var Resolver = net.DefaultResolver

// DialContext returns a dial function that resolves the host itself, checks every address
// against the policy and connects to a checked address (never re-resolving).
func (p Policy) DialContext(timeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		var ips []netip.Addr
		if a, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil {
			ips = []netip.Addr{a}
		} else {
			res, err := Resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			ips = res
		}
		var lastErr error = fmt.Errorf("%w: %s resolves to nothing", ErrRefused, host)
		for _, ip := range ips {
			if !p.allows(host, port, ip) {
				lastErr = fmt.Errorf("%w: %s (%s)", ErrRefused, host, ip)
				continue
			}
			c, err := d.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
			if err == nil {
				return c, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

// Control is a net.Dialer Control func applying the policy to the connected address (for
// libraries that only accept a *net.Dialer). It cannot see the host name, so name-based
// allow-list entries do not apply.
func (p Policy) Control(_ string, address string, _ syscall.RawConn) error {
	ap, err := netip.ParseAddrPort(address)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrRefused, address)
	}
	if !p.allows(ap.Addr().String(), strconv.Itoa(int(ap.Port())), ap.Addr()) {
		return fmt.Errorf("%w: %s", ErrRefused, address)
	}
	return nil
}

// Client returns an HTTP client dialing through the policy. maxRedirects < 0 never follows
// redirects (the last answer is returned); otherwise each hop is dialed through the policy too.
func (p Policy) Client(timeout time.Duration, maxRedirects int) *http.Client {
	tr := &http.Transport{
		Proxy:                 nil, // a proxy would dial on our behalf, out of the policy
		DialContext:           p.DialContext(10 * time.Second),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
	c := &http.Client{Timeout: timeout, Transport: tr}
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if maxRedirects < 0 {
			return http.ErrUseLastResponse
		}
		if len(via) > maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("%w: redirect to %s", ErrRefused, req.URL.Scheme)
		}
		return nil
	}
	return c
}

// Class is a coarse, safe description of an outbound failure: it can be shown to the person who
// triggered the request without turning the server into a port scanner.
type Class string

const (
	ClassRefused     Class = "refused_by_policy"
	ClassTimeout     Class = "timeout"
	ClassUnreachable Class = "unreachable"
)

// Classify maps a transport error to a Class.
func Classify(err error) Class {
	var ne net.Error
	switch {
	case errors.Is(err, ErrRefused):
		return ClassRefused
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &ne) && ne.Timeout():
		return ClassTimeout
	}
	return ClassUnreachable
}

// HostPort returns the lower-cased host and port of an http(s) URL, the port defaulted by scheme.
func HostPort(raw string) (host, port string, ok bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", "", false
	}
	port = u.Port()
	if port == "" {
		port = "80"
		if u.Scheme == "https" {
			port = "443"
		}
	}
	return strings.ToLower(u.Hostname()), port, true
}

// Resolve returns the addresses of host (a literal gives itself).
func Resolve(ctx context.Context, host string) []netip.Addr {
	if a, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{a.Unmap()}
	}
	res, err := Resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil
	}
	for i := range res {
		res[i] = res[i].Unmap()
	}
	return res
}

// Allowed reports whether the policy lets rawURL through, resolving its host now (used to warn
// about saved webhooks; deliveries are checked at dial time).
func (p Policy) Allowed(ctx context.Context, rawURL string) bool {
	host, port, ok := HostPort(rawURL)
	if !ok {
		return false
	}
	ips := Resolve(ctx, host)
	if len(ips) == 0 {
		return true // unknown now: decided at delivery
	}
	for _, ip := range ips {
		if p.allows(host, port, ip) {
			return true
		}
	}
	return false
}
