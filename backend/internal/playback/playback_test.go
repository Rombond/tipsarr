package playback

import "testing"

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
