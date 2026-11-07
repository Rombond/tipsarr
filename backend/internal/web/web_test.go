package web

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestScriptHashes(t *testing.T) {
	body := "\n\t\t\tconsole.log(1);\n\t\t"
	sum := sha256.Sum256([]byte(body))
	want := "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	got := scriptHashes([]byte(`<head><script>` + body + `</script><script src="/x.js"></script><script>a()</script></head>`))
	if len(got) != 2 || got[0] != want {
		t.Fatalf("got %v, want first %s", got, want)
	}
}
