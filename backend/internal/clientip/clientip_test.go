package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	r, err := Parse(DefaultTrusted)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, peer, xff, xreal, tci, want string
	}{
		{"direct, headers ignored", "203.0.113.5:1", "1.2.3.4", "1.2.3.4", "1.2.3.4", "203.0.113.5"},
		{"proxy, xff", "172.18.0.2:1", "198.51.100.7", "", "", "198.51.100.7"},
		{"proxy, forged left part", "172.18.0.2:1", "6.6.6.6, 198.51.100.7", "", "", "198.51.100.7"},
		{"proxy chain", "127.0.0.1:1", "198.51.100.7, 10.0.0.3", "", "", "198.51.100.7"},
		{"proxy, x-real-ip", "127.0.0.1:1", "", "198.51.100.8", "", "198.51.100.8"},
		{"true-client-ip ignored", "127.0.0.1:1", "", "", "9.9.9.9", "127.0.0.1"},
		{"all trusted", "127.0.0.1:1", "192.168.1.20", "", "", "192.168.1.20"},
		{"garbage", "127.0.0.1:1", "nope", "", "", "127.0.0.1"},
		{"ipv6 peer", "[::1]:1", "2001:db8::1", "", "", "2001:db8::1"},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = c.peer
		if c.xff != "" {
			req.Header.Set("X-Forwarded-For", c.xff)
		}
		if c.xreal != "" {
			req.Header.Set("X-Real-IP", c.xreal)
		}
		if c.tci != "" {
			req.Header.Set("True-Client-IP", c.tci)
		}
		if got := r.ClientIP(req); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	none, _ := Parse("")
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	if got := none.ClientIP(req); got != "127.0.0.1" {
		t.Errorf("empty list must trust nobody, got %s", got)
	}
	if _, err := Parse("10.0.0.0/8, bogus"); err == nil {
		t.Error("bad entry must fail")
	}
}
