package playback

import (
	"context"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/media"
)

func TestNormaliseMatchesTitleSpellings(t *testing.T) {
	same := [][2]string{
		{"Pacific Rim : Uprising", "Pacific Rim: Uprising"},
		{"Spider-Man : No Way Home", "Spider-Man: No Way Home"},
		{"X-Men", "X‑Men"},
		{"Godzilla vs. Kong", "godzilla vs kong"},
	}
	for _, p := range same {
		if normalise(p[0]) != normalise(p[1]) {
			t.Errorf("%q and %q should match", p[0], p[1])
		}
	}
	if normalise("Pacific Rim") == normalise("Pacific Rim : Uprising") {
		t.Error("a sequel is not the same title")
	}
}

type fakeTMDB map[string][]media.Item // language -> results of any search

func (f fakeTMDB) SearchTitle(_ context.Context, _, lang, _ string) ([]media.Item, error) {
	return f[lang], nil
}
func (f fakeTMDB) Detail(context.Context, media.Opts, string, int) (*media.Detail, error) {
	return &media.Detail{}, nil
}

// "Là-haut" is "Up" in French but the title of another, older film in English: the app's language
// is tried first, and a result without a poster is not trusted.
func TestMatchPrefersTheAppLanguageAndAPoster(t *testing.T) {
	svc := &Service{tmdb: fakeTMDB{
		"":      {{TMDBID: 146952, Title: "Là-haut", ReleaseDate: "2004-01-01"}},
		"fr-FR": {{TMDBID: 14160, Title: "Là-haut", PosterPath: "/up.jpg", ReleaseDate: "2009-05-13"}},
	}}
	hit, err := svc.match(context.Background(), "movie", "Là-haut", "fr-FR")
	if err != nil || hit == nil || hit.TMDBID != 14160 {
		t.Fatalf("French app: %+v %v", hit, err)
	}
	// English app: only the poster-less 2004 entry exists, so nothing is claimed
	hit, err = svc.match(context.Background(), "movie", "Là-haut", "")
	if err != nil || hit != nil {
		t.Fatalf("English app: %+v %v", hit, err)
	}
}
