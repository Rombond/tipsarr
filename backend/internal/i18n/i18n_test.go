package i18n

import "testing"

func TestMessageTextAndFallbacks(t *testing.T) {
	m := M("srv.job.library.synced", "movies", 12, "shows", 3, "episodes", 40, "unmatched", 2)
	if got, want := m.String(), "12 movies, 3 TV shows, 40 episodes (2 without a TMDB id)"; got != want {
		t.Errorf("English = %q, want %q", got, want)
	}
	if got, _ := T("fr-CA", m.Code, m.Params); got != "12 films, 3 séries, 40 épisodes (2 sans identifiant TMDB)" {
		t.Errorf("French = %q", got)
	}
	if got, ok := T("xx", m.Code, m.Params); !ok || got != m.String() {
		t.Errorf("unknown language should fall back to English, got %q %v", got, ok)
	}
	if _, ok := T("en", "no.such.code", nil); ok {
		t.Error("unknown code reported as known")
	}
	if !(Msg{}).IsZero() || (Msg{}).String() != "" {
		t.Error("zero message")
	}
	// a placeholder without a value is left visible, not dropped
	if got := M("srv.job.import.done").String(); got != "{count} imported" {
		t.Errorf("missing param = %q", got)
	}
}

// Every language has the same placeholders as English for every server text (the locale checker enforces it too).
func TestEveryServerTextHasAnEnglishSource(t *testing.T) {
	for lang, texts := range catalogs {
		for code := range texts {
			if _, ok := catalogs["en"][code]; !ok {
				t.Errorf("%s has %s, English does not", lang, code)
			}
		}
	}
}

func TestOrderedArgsFollowTheEnglishPlaceholders(t *testing.T) {
	m := M("srv.job.library.synced", "unmatched", 2, "episodes", 40, "shows", 3, "movies", 12)
	got := m.OrderedArgs()
	want := []string{"12", "3", "40", "2"} // movies, shows, episodes, unmatched
	if len(got) != len(want) {
		t.Fatalf("args = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args = %v, want %v", got, want)
		}
	}
	if M("srv.job.housekeeping.purged").OrderedArgs() != nil {
		t.Error("a message without placeholders has no args")
	}
}
