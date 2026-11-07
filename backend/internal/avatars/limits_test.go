package avatars

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"testing"
)

// pngHeader is a PNG that only has a header claiming w x h (decoding it fully would fail).
func pngHeader(w, h uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestNormaliseRefusesHugeCanvas(t *testing.T) {
	for _, d := range [][2]uint32{{40000, 40000}, {4097, 10}, {4096, 4096 + 1}} {
		if _, err := normalise(pngHeader(d[0], d[1])); !errors.Is(err, ErrTooBig) {
			t.Errorf("%v: err = %v", d, err)
		}
	}
	if _, err := normalise(make([]byte, maxBytes+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("oversized body: %v", err)
	}
	var ok bytes.Buffer
	_ = png.Encode(&ok, image.NewRGBA(image.Rect(0, 0, 300, 200)))
	if _, err := normalise(ok.Bytes()); err != nil {
		t.Fatalf("small picture: %v", err)
	}
}

func TestLDAPFailureIsGeneric(t *testing.T) {
	err := CheckLDAP(context.Background(), "ldap://127.0.0.1:1", "cn=x", "y")
	if txt := LDAPFailure(err); txt != "the server could not be reached" {
		t.Fatalf("%v -> %q", err, txt)
	}
	if LDAPFailure(CheckLDAP(context.Background(), "http://x", "", "")) != errBadLDAPURL.Error() {
		t.Fatal("bad url")
	}
}
