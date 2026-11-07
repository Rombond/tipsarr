// Package clientip finds the address of the person behind a request, trusting forwarding headers only
// when they come from a known reverse proxy.
package clientip

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// DefaultTrusted are the private and loopback ranges: a reverse proxy on the same host or LAN (Caddy,
// Traefik, nginx, a Docker network, the demo gateway) is trusted out of the box.
const DefaultTrusted = "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,127.0.0.0/8,::1/128,fc00::/7"

// Resolver holds the trusted proxy ranges.
type Resolver struct{ trusted []netip.Prefix }

// Parse reads a comma or space separated list of CIDRs or single addresses. An empty list trusts nobody.
func Parse(list string) (*Resolver, error) {
	r := &Resolver{}
	for _, f := range strings.FieldsFunc(list, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' }) {
		if p, err := netip.ParsePrefix(f); err == nil {
			r.trusted = append(r.trusted, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(f)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q is neither an address nor a CIDR", f)
		}
		r.trusted = append(r.trusted, netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen()))
	}
	return r, nil
}

func (r *Resolver) isTrusted(s string) bool {
	a, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return false
	}
	a = a.Unmap()
	for _, p := range r.trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// ClientIP is the visitor's address. The peer address is used unless the peer is a trusted proxy; then
// X-Forwarded-For is read from the right, skipping trusted hops, and the first untrusted address wins (the
// left part can be forged by the client). Without X-Forwarded-For, X-Real-IP is used. True-Client-IP and
// other headers are ignored.
func (r *Resolver) ClientIP(req *http.Request) string {
	peer, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		peer = req.RemoteAddr
	}
	if !r.isTrusted(peer) {
		return peer
	}
	if xff := req.Header.Values("X-Forwarded-For"); len(xff) > 0 {
		hops := strings.Split(strings.Join(xff, ","), ",")
		last := ""
		for i := len(hops) - 1; i >= 0; i-- {
			h := strings.TrimSpace(hops[i])
			if _, err := netip.ParseAddr(h); err != nil {
				break // garbage: stop at what we could read
			}
			last = h
			if !r.isTrusted(h) {
				return h
			}
		}
		if last != "" {
			return last // every hop is trusted: the leftmost valid one is the client
		}
	}
	if xr := strings.TrimSpace(req.Header.Get("X-Real-IP")); xr != "" {
		if _, err := netip.ParseAddr(xr); err == nil {
			return xr
		}
	}
	return peer
}

// Middleware rewrites r.RemoteAddr to the resolved client address (port dropped), like chi's RealIP but
// only for trusted proxies.
func (r *Resolver) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ip := r.ClientIP(req); ip != "" {
			req.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		next.ServeHTTP(w, req)
	})
}
