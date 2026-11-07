package fakejf

import (
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"strings"
)

// 5x7 block font: seven rows of five bits per glyph.
var font = map[rune]string{
	'A': "01110,10001,10001,11111,10001,10001,10001", 'B': "11110,10001,10001,11110,10001,10001,11110", 'C': "01110,10001,10000,10000,10000,10001,01110",
	'D': "11110,10001,10001,10001,10001,10001,11110", 'E': "11111,10000,10000,11110,10000,10000,11111", 'F': "11111,10000,10000,11110,10000,10000,10000",
	'G': "01110,10001,10000,10111,10001,10001,01111", 'H': "10001,10001,10001,11111,10001,10001,10001", 'I': "01110,00100,00100,00100,00100,00100,01110",
	'J': "00111,00010,00010,00010,00010,10010,01100", 'K': "10001,10010,10100,11000,10100,10010,10001", 'L': "10000,10000,10000,10000,10000,10000,11111",
	'M': "10001,11011,10101,10101,10001,10001,10001", 'N': "10001,11001,10101,10011,10001,10001,10001", 'O': "01110,10001,10001,10001,10001,10001,01110",
	'P': "11110,10001,10001,11110,10000,10000,10000", 'Q': "01110,10001,10001,10001,10101,10010,01101", 'R': "11110,10001,10001,11110,10100,10010,10001",
	'S': "01111,10000,10000,01110,00001,00001,11110", 'T': "11111,00100,00100,00100,00100,00100,00100", 'U': "10001,10001,10001,10001,10001,10001,01110",
	'V': "10001,10001,10001,10001,10001,01010,00100", 'W': "10001,10001,10001,10101,10101,11011,10001", 'X': "10001,10001,01010,00100,01010,10001,10001",
	'Y': "10001,10001,01010,00100,00100,00100,00100", 'Z': "11111,00001,00010,00100,01000,10000,11111",
	'0': "01110,10001,10011,10101,11001,10001,01110", '1': "00100,01100,00100,00100,00100,00100,01110", '2': "01110,10001,00001,00010,00100,01000,11111",
	'3': "11110,00001,00001,01110,00001,00001,11110", '4': "00010,00110,01010,10010,11111,00010,00010", '5': "11111,10000,11110,00001,00001,10001,01110",
	'6': "00110,01000,10000,11110,10001,10001,01110", '7': "11111,00001,00010,00100,01000,01000,01000", '8': "01110,10001,10001,01110,10001,10001,01110",
	'9': "01110,10001,10001,01111,00001,00010,01100", '-': "00000,00000,00000,11111,00000,00000,00000", ':': "00000,00100,00000,00000,00000,00100,00000",
	'\'': "00100,00100,00000,00000,00000,00000,00000", ' ': "00000,00000,00000,00000,00000,00000,00000",
}

func hsv(h, s, v float64) color.RGBA {
	i := int(h*6) % 6
	f := h*6 - float64(int(h*6))
	p, q, t := v*(1-s), v*(1-f*s), v*(1-(1-f)*s)
	var r, g, b float64
	switch i {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), 255}
}

func drawText(img *image.RGBA, text string, x0, y0, scale int, c color.RGBA, maxW int) int {
	x, y := x0, y0
	adv := 6 * scale
	for _, word := range strings.Fields(strings.ToUpper(text)) {
		if x+len(word)*adv > x0+maxW && x != x0 {
			x, y = x0, y+9*scale
		}
		for _, r := range word + " " {
			g, ok := font[r]
			if !ok {
				g = font[' ']
			}
			for row, bits := range strings.Split(g, ",") {
				for col, b := range bits {
					if b != '1' {
						continue
					}
					for dy := 0; dy < scale; dy++ {
						for dx := 0; dx < scale; dx++ {
							px, py := x+col*scale+dx, y+row*scale+dy
							if image.Pt(px, py).In(img.Bounds()) {
								img.SetRGBA(px, py, c)
							}
						}
					}
				}
			}
			x += adv
		}
	}
	return y + 9*scale
}

// Poster draws a gradient with the title and id in block letters.
func Poster(w, h int, hue float64, label, sub string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		v := 0.85 - 0.5*float64(y)/float64(h)
		c := hsv(hue, 0.65, v)
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	scale := max(w/90, 1)
	y := drawText(img, label, w/12, h/3, scale*2, color.RGBA{255, 255, 255, 255}, w-w/6)
	drawText(img, sub, w/12, y+scale*3, scale, color.RGBA{255, 255, 255, 200}, w-w/6)
	return img
}

// WriteJPEG answers with the image as a JPEG.
func WriteJPEG(w http.ResponseWriter, img image.Image) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_ = jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
}
