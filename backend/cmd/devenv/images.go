package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/fakejf"
)

// artFor resolves an image file name such as m12.jpg, bt210.jpg or pp503.jpg into a title and a ratio (height / width).
func artFor(name string) (label, sub string, ratio float64, hue float64) {
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".jpg"), "-fake")
	prefix := strings.TrimRight(name, "0123456789")
	id, _ := strconv.Atoi(strings.TrimPrefix(name, prefix))
	ratio = 1.5
	switch prefix {
	case "b", "bt", "s":
		ratio = 0.5625
	case "pp":
		return personName(id), "ACTOR", 1.5, float64(id%20) / 20
	}
	if id >= 701 && id <= 703 {
		return "SAGA " + strconv.Itoa(id-700), "COLLECTION", ratio, float64(id%20) / 20
	}
	kind := "MOVIE"
	if isShow(id) {
		kind = "TV " + strconv.Itoa(id)
	} else {
		kind += " " + strconv.Itoa(id)
	}
	return title(id, "en"), kind, ratio, float64((id*7)%20) / 20
}

func newImages() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			http.NotFound(w, r)
			return
		}
		width := 342
		if n, err := strconv.Atoi(strings.TrimPrefix(parts[0], "w")); err == nil {
			width = n
		}
		if parts[0] == "original" {
			width = 780
		}
		label, sub, ratio, hue := artFor(parts[len(parts)-1])
		fakejf.WriteJPEG(w, fakejf.Poster(width, int(float64(width)*ratio), hue, label, sub))
	})
}
