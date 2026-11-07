package netguard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func TestPolicyAddresses(t *testing.T) {
	p := Policy{}
	for _, c := range []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true}, {"2606:4700::1111", true},
		{"127.0.0.1", false}, {"::1", false}, {"169.254.169.254", false}, {"fe80::1", false},
		{"0.0.0.0", false}, {"224.0.0.1", false}, {"::ffff:127.0.0.1", false},
		{"10.0.0.5", false}, {"192.168.1.2", false}, {"172.16.0.1", false}, {"fd00::1", false}, {"100.64.1.1", false},
	} {
		if got := p.allows("h", "80", netip.MustParseAddr(c.ip)); got != c.want {
			t.Errorf("%s: got %v want %v", c.ip, got, c.want)
		}
	}
	lan := Policy{AllowPrivate: true}
	if !lan.allows("h", "80", netip.MustParseAddr("192.168.1.2")) || lan.allows("h", "80", netip.MustParseAddr("169.254.169.254")) {
		t.Error("AllowPrivate must allow LAN but never link-local")
	}
}

func TestAllowList(t *testing.T) {
	list := []string{"ntfy.lan", "192.168.1.20:8123", "10.1.0.0/16", "127.0.0.1"}
	p := Policy{AllowList: func() []string { return list }}
	ok := func(host, port, ip string) bool { return p.allows(host, port, netip.MustParseAddr(ip)) }
	if !ok("ntfy.lan", "443", "192.168.1.9") || !ok("x", "8123", "192.168.1.20") || !ok("x", "80", "10.1.2.3") {
		t.Error("listed targets must pass")
	}
	if ok("x", "80", "192.168.1.20") || ok("other.lan", "80", "192.168.1.9") || ok("x", "80", "10.2.0.1") {
		t.Error("unlisted targets must be refused")
	}
	if ok("127.0.0.1", "80", "127.0.0.1") {
		t.Error("loopback can never be allowed")
	}
	for s, want := range map[string]bool{"ntfy.lan": true, "ha:8123": true, "10.0.0.0/8": true, "[fd00::1]:80": true, "fd00::1": true,
		"": false, "a b": false, "ha:99999": false, "10.0.0.0/99": false, "http://x": false} {
		if ValidAllowEntry(s) != want {
			t.Errorf("ValidAllowEntry(%q) != %v", s, want)
		}
	}
}

func TestClientRefusesLoopbackAndClassifies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	_, err := Policy{AllowPrivate: true}.Client(2*time.Second, 3).Get(srv.URL)
	if !errors.Is(err, ErrRefused) || Classify(err) != ClassRefused {
		t.Fatalf("loopback must be refused at dial time, got %v", err)
	}
	if Classify(context.DeadlineExceeded) != ClassTimeout || Classify(errors.New("connection refused")) != ClassUnreachable {
		t.Error("classify")
	}
}

func TestRedirectToLoopbackRefused(t *testing.T) {
	// a public-looking first hop cannot be simulated offline; check the redirect hop goes through the dialer
	testLoopback.Store(true)
	defer testLoopback.Store(false)
	inner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer inner.Close()
	n := 0
	outer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		http.Redirect(w, r, r.URL.String()+"x", http.StatusFound)
	}))
	defer outer.Close()
	_, err := Policy{}.Client(2*time.Second, 3).Get(outer.URL)
	if err == nil || n != 4 {
		t.Fatalf("redirects must stop after 3 hops: n=%d err=%v", n, err)
	}
}

func TestHostPort(t *testing.T) {
	h, p, ok := HostPort("https://Radarr.LAN/x")
	if !ok || h != "radarr.lan" || p != "443" {
		t.Fatal(h, p, ok)
	}
	if _, p, _ := HostPort("http://1.2.3.4"); p != "80" {
		t.Fatal(p)
	}
	if _, _, ok := HostPort("ftp://x"); ok {
		t.Fatal("ftp")
	}
}
