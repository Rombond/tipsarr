package avatars

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// withOrientation inserts an EXIF APP1 segment carrying the orientation after the JPEG start marker.
func withOrientation(jpg []byte, o uint16, bigEndian bool) []byte {
	var bo binary.AppendByteOrder = binary.LittleEndian
	tag := "II"
	if bigEndian {
		bo, tag = binary.BigEndian, "MM"
	}
	t := []byte(tag)
	t = bo.AppendUint16(t, 42)
	t = bo.AppendUint32(t, 8)
	t = bo.AppendUint16(t, 1) // one entry
	t = bo.AppendUint16(t, 0x0112)
	t = bo.AppendUint16(t, 3) // SHORT
	t = bo.AppendUint32(t, 1)
	t = bo.AppendUint16(t, o)
	t = append(t, 0, 0)
	t = bo.AppendUint32(t, 0) // no next IFD
	seg := append([]byte("Exif\x00\x00"), t...)
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1}
	out = binary.BigEndian.AppendUint16(out, uint16(len(seg)+2))
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func sample(t *testing.T) []byte {
	t.Helper()
	// 40 x 20: left half red, right half blue
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			c := color.RGBA{R: 255, A: 255}
			if x >= 20 {
				c = color.RGBA{B: 255, A: 255}
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExifOrientationIsRead(t *testing.T) {
	base := sample(t)
	if got := exifOrientation(base); got != 1 {
		t.Fatalf("no exif = %d", got)
	}
	for _, be := range []bool{false, true} {
		if got := exifOrientation(withOrientation(base, 6, be)); got != 6 {
			t.Fatalf("orientation 6 (bigEndian=%v) = %d", be, got)
		}
	}
	if got := exifOrientation([]byte("not a jpeg")); got != 1 {
		t.Fatalf("garbage = %d", got)
	}
}

// A phone photo that says "turn 90 clockwise" must come out upright, not sideways.
func TestNormaliseTurnsSidewaysPhotos(t *testing.T) {
	left := func(img image.Image) (r, b uint32) {
		r1, _, b1, _ := img.At(10, 128).RGBA()
		r2, _, b2, _ := img.At(246, 128).RGBA()
		_, _ = r2, b2
		return r1 >> 8, b1 >> 8
	}
	decode := func(raw []byte) image.Image {
		img, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	// without the tag the square keeps red on the left
	plain, err := normalise(sample(t))
	if err != nil {
		t.Fatal(err)
	}
	if r, b := left(decode(plain)); r < 200 || b > 60 {
		t.Fatalf("untagged: left pixel r=%d b=%d", r, b)
	}
	// orientation 6: the photo's left side ends up on top, so the red half is at the top
	turned, err := normalise(withOrientation(sample(t), 6, false))
	if err != nil {
		t.Fatal(err)
	}
	img := decode(turned)
	rTop, _, _, _ := img.At(128, 10).RGBA()
	_, _, bBottom, _ := img.At(128, 246).RGBA()
	if rTop>>8 < 200 || bBottom>>8 < 200 {
		t.Fatalf("orientation 6: top r=%d bottom b=%d, want red on top and blue at the bottom", rTop>>8, bBottom>>8)
	}
}
