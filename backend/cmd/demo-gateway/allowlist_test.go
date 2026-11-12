package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestAllowlist(t *testing.T) {
	a := ParseAllowlist("127.0.0.1/8 ::1, 192.168.2.0/24 203.0.113.7 home.example", time.Minute)
	ip := "198.51.100.5"
	fail := false
	a.lookup = func(context.Context, string) ([]net.IP, error) {
		if fail {
			return nil, errors.New("dns down")
		}
		return []net.IP{net.ParseIP(ip)}, nil
	}
	now := time.Unix(1000, 0)
	a.now = func() time.Time { return now }
	for addr, want := range map[string]bool{"127.0.0.5": true, "::1": true, "192.168.2.40": true, "203.0.113.7": true,
		"198.51.100.5": true, "203.0.113.8": false, "bad": false} {
		if got := a.Allowed(addr); got != want {
			t.Errorf("Allowed(%q) = %v, want %v", addr, got, want)
		}
	}
	// the host moves; a failed lookup keeps the old address, a later one picks up the new one
	ip, fail, now = "198.51.100.9", true, now.Add(2*time.Minute)
	if !a.Allowed("198.51.100.5") {
		t.Error("failed lookup dropped the previous address")
	}
	fail, now = false, now.Add(2*time.Minute)
	if !a.Allowed("198.51.100.9") || a.Allowed("198.51.100.5") {
		t.Error("refresh did not follow the new address")
	}
	if !ParseAllowlist("", time.Minute).Empty() || (*Allowlist)(nil).Allowed("127.0.0.1") {
		t.Error("empty list must allow nothing")
	}
}
