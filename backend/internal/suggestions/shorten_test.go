package suggestions

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestShortenCutsAtWordBoundary(t *testing.T) {
	if got := shorten("short text", 400); got != "short text" {
		t.Fatalf("short text changed: %q", got)
	}
	long := strings.Repeat("word ", 200)
	got := shorten(long, 100)
	if len(got) > 104 || !strings.HasSuffix(got, "word…") {
		t.Fatalf("got %q (len %d)", got, len(got))
	}
	// multi-byte text must never be split mid-character
	jp := strings.Repeat("日本語のテキスト", 100)
	cut := shorten(jp, 100)
	if !utf8.ValidString(cut) || !strings.HasSuffix(cut, "…") {
		t.Fatalf("invalid utf-8 or no ellipsis: %q", cut)
	}
}
