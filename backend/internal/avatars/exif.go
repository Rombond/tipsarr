package avatars

import (
	"encoding/binary"
	"image"
)

// exifOrientation reads the EXIF orientation (1-8) of a JPEG, or 1 when there is none. Phones
// store photos sideways and say so in this tag; decoding and re-encoding drops the tag, so the
// picture has to be turned by hand or it shows up rotated.
func exifOrientation(raw []byte) int {
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return 1
	}
	for i := 2; i+4 <= len(raw); {
		if raw[i] != 0xFF {
			return 1
		}
		marker := raw[i+1]
		if marker == 0xFF { // padding
			i++
			continue
		}
		if marker == 0xD9 || marker == 0xDA { // end of image / start of pixel data
			return 1
		}
		size := int(binary.BigEndian.Uint16(raw[i+2:]))
		if size < 2 || i+2+size > len(raw) {
			return 1
		}
		if marker == 0xE1 && size >= 8 && string(raw[i+4:i+10]) == "Exif\x00\x00" {
			return tiffOrientation(raw[i+10 : i+2+size])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:]))
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[ifd:]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + 12*k
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 { // Orientation, a SHORT
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// orient turns an image so that it is upright, given its EXIF orientation.
func orient(src *image.RGBA, o int) *image.RGBA {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 { // 90 degree turns swap the sides
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2: // mirrored horizontally
				nx, ny = w-1-x, y
			case 3: // turned 180
				nx, ny = w-1-x, h-1-y
			case 4: // mirrored vertically
				nx, ny = x, h-1-y
			case 5: // transposed
				nx, ny = y, x
			case 6: // needs 90 clockwise
				nx, ny = h-1-y, x
			case 7: // transversed
				nx, ny = h-1-y, w-1-x
			case 8: // needs 90 counter-clockwise
				nx, ny = y, w-1-x
			}
			dst.SetRGBA(nx, ny, src.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
