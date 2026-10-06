package tmdb

import (
	"context"
	"strings"
	"testing"
)

// A transport error must never carry the API key (v3 keys travel in the query string).
func TestErrorsDoNotLeakTheKey(t *testing.T) {
	c := New("http://127.0.0.1:1", "SECRETKEY1234567890")
	_, err := c.Get(context.Background(), "/trending/all/week", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "SECRETKEY") || strings.Contains(err.Error(), "api_key") {
		t.Fatalf("error leaks the key: %v", err)
	}
}
