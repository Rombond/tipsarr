package main

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// Allowlist holds client addresses exempt from the per-IP session limit (DEMO_IP_ALLOWLIST): IPs, CIDRs and host
// names, separated by spaces or commas, the same syntax as fail2ban's ignoreip. Host names (dynamic DNS) are resolved
// again every Refresh, so a changing home address keeps working.
type Allowlist struct {
	nets    []*net.IPNet
	hosts   []string
	refresh time.Duration
	lookup  func(ctx context.Context, host string) ([]net.IP, error)
	now     func() time.Time

	mu       sync.Mutex
	resolved map[string]bool
	at       time.Time
}

// ParseAllowlist parses the list; an entry that is neither an IP nor a CIDR is kept as a host name.
func ParseAllowlist(s string, refresh time.Duration) *Allowlist {
	a := &Allowlist{refresh: refresh, now: time.Now, lookup: func(ctx context.Context, host string) ([]net.IP, error) {
		return net.DefaultResolver.LookupIP(ctx, "ip", host)
	}}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' }) {
		if _, n, err := net.ParseCIDR(f); err == nil {
			a.nets = append(a.nets, n)
		} else if ip := net.ParseIP(f); ip != nil {
			bits := 128
			if ip.To4() != nil {
				ip, bits = ip.To4(), 32
			}
			a.nets = append(a.nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		} else {
			a.hosts = append(a.hosts, f)
		}
	}
	return a
}

// Empty reports whether the list has no entry.
func (a *Allowlist) Empty() bool { return a == nil || len(a.nets)+len(a.hosts) == 0 }

// Allowed reports whether ip is on the list.
func (a *Allowlist) Allowed(ip string) bool {
	if a.Empty() {
		return false
	}
	p := net.ParseIP(ip)
	if p == nil {
		return false
	}
	for _, n := range a.nets {
		if n.Contains(p) {
			return true
		}
	}
	if len(a.hosts) == 0 {
		return false
	}
	return a.hostIPs()[p.String()]
}

// hostIPs resolves the host names, at most once per refresh period; a failed lookup keeps the previous addresses.
func (a *Allowlist) hostIPs() map[string]bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.resolved != nil && a.now().Sub(a.at) < a.refresh {
		return a.resolved
	}
	next := map[string]bool{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, h := range a.hosts {
		ips, err := a.lookup(ctx, h)
		if err != nil {
			for k := range a.resolved {
				next[k] = true
			}
			continue
		}
		for _, ip := range ips {
			next[ip.String()] = true
		}
	}
	a.resolved, a.at = next, a.now()
	return next
}
