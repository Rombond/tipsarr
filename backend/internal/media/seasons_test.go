package media

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

var bleachTMDB = []Season{
	{Number: 0, Name: "Specials", EpisodeCount: 4},
	{Number: 1, Name: "Bleach", EpisodeCount: 366, AirDate: "2004-10-05", PosterPath: "/s1.jpg"},
	{Number: 2, Name: "Thousand-Year Blood War", EpisodeCount: 50, AirDate: "2022-10-11", PosterPath: "/s2.jpg"},
}

var bleachCounts = []int{7, 20, 21, 22, 28, 18, 22, 20, 16, 22, 16, 7, 17, 36, 51, 26, 24, 50}

func bleachTVDB(withCounts bool) []TVDBSeason {
	var out []TVDBSeason
	for n, c := range bleachCounts {
		s := TVDBSeason{Number: n}
		if withCounts {
			s.EpisodeCount = c
			if n == 17 {
				s.FirstAired = "2022-10-11"
			} else if n == 1 {
				s.FirstAired = "2004-10-05"
			}
		}
		out = append(out, s)
	}
	return out
}

func testName(n int) string { return "Season " + strconv.Itoa(n) }

func TestNumberingDiffers(t *testing.T) {
	tvdb := func(ns ...int) []TVDBSeason {
		var out []TVDBSeason
		for _, n := range ns {
			out = append(out, TVDBSeason{Number: n})
		}
		return out
	}
	tm := func(counts ...int) []Season {
		var out []Season
		for i, c := range counts {
			out = append(out, Season{Number: i + 1, EpisodeCount: c})
		}
		return out
	}
	cases := []struct {
		name string
		tmdb []Season
		tvdb []TVDBSeason
		want bool
	}{
		{"bleach", bleachTMDB, bleachTVDB(true), true},
		{"bleach lookup only", bleachTMDB, bleachTVDB(false), true},
		{"witch hat new season", tm(13), tvdb(1, 2), false},
		{"equal", tm(10, 10), tvdb(1, 2), false},
		{"gap", tm(10, 10), tvdb(1, 3), true},
		{"counts off", tm(10, 30), []TVDBSeason{{Number: 1, EpisodeCount: 10}, {Number: 2, EpisodeCount: 20}}, true},
		{"counts close", tm(10, 22), []TVDBSeason{{Number: 1, EpisodeCount: 10}, {Number: 2, EpisodeCount: 20}}, false},
		{"empty tvdb", tm(10), nil, false},
	}
	for _, c := range cases {
		if got := NumberingDiffers(c.tmdb, c.tvdb); got != c.want {
			t.Errorf("%s: differs = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMergeSeasonsBleach(t *testing.T) {
	out, differs := MergeSeasons(bleachTMDB, bleachTVDB(true), testName)
	if !differs || len(out) != 18 {
		t.Fatalf("differs=%v len=%d", differs, len(out))
	}
	s17 := out[17]
	if s17.Number != 17 || s17.TMDBNumber != 2 || s17.Name != "Thousand-Year Blood War" || s17.EpisodeCount != 50 || s17.PosterPath != "/s2.jpg" {
		t.Fatalf("S17 = %+v", s17)
	}
	s1 := out[1]
	if s1.TMDBNumber != 0 || s1.EpisodeCount != 20 || s1.Name != "Season 1" || s1.PosterPath != "" || s1.CountUnknown {
		t.Fatalf("S1 = %+v", s1)
	}
	if out[0].TMDBNumber != 0 || out[0].EpisodeCount != 7 {
		t.Fatalf("S0 = %+v", out[0])
	}
}

func TestMergeSeasonsLookupOnly(t *testing.T) {
	out, differs := MergeSeasons(bleachTMDB, bleachTVDB(false), testName)
	if !differs || len(out) != 18 {
		t.Fatalf("differs=%v len=%d", differs, len(out))
	}
	for _, s := range out[1:] {
		if s.Name != testName(s.Number) || !s.CountUnknown || s.EpisodeCount != 0 || s.TMDBNumber != 0 {
			t.Fatalf("season %+v", s)
		}
	}
}

func TestMergeSeasonsAligned(t *testing.T) {
	tmdb := []Season{{Number: 1, Name: "Pilot season", EpisodeCount: 13, AirDate: "2020-01-01", PosterPath: "/p.jpg"}}
	out, differs := MergeSeasons(tmdb, []TVDBSeason{{Number: 1, Monitored: true}, {Number: 2}}, testName)
	if differs || len(out) != 2 {
		t.Fatalf("differs=%v out=%+v", differs, out)
	}
	if out[0].Name != "Pilot season" || out[0].EpisodeCount != 13 || out[0].TMDBNumber != 1 || !out[0].Monitored || out[0].CountUnknown {
		t.Fatalf("S1 = %+v", out[0])
	}
	if out[1].Name != "Season 2" || !out[1].CountUnknown {
		t.Fatalf("S2 = %+v", out[1])
	}
}

func TestSeasonNameLocalized(t *testing.T) {
	if got := SeasonName("fr-FR")(3); got != "Saison 3" {
		t.Fatalf("fr = %q", got)
	}
	if got := SeasonName("en")(0); got != "Specials" {
		t.Fatalf("en specials = %q", got)
	}
}

func TestLibraryAvailabilityUnknownCounts(t *testing.T) {
	seasons := []Season{{Number: 1, CountUnknown: true, AirDate: "2004-10-05"}, {Number: 2, EpisodeCount: 10}}
	if got := libraryAvailability(seasons, map[int]int{1: 3, 2: 10}); got != AvailabilityAvailable {
		t.Fatalf("got %s", got)
	}
	if seasons[0].LibraryEpisodes != 3 {
		t.Fatal("library episodes not set")
	}
	if got := libraryAvailability(seasons, map[int]int{2: 10}); got != AvailabilityPartial {
		t.Fatalf("aired unknown season missing: got %s", got)
	}
	future := []Season{{Number: 1, EpisodeCount: 10}, {Number: 2, CountUnknown: true}}
	if got := libraryAvailability(future, map[int]int{1: 10}); got != AvailabilityAvailable {
		t.Fatalf("undated unknown season must not count: got %s", got)
	}
}

type fakeSource struct {
	seasons []TVDBSeason
	ok      bool
	calls   int
}

func (f *fakeSource) TVDBSeasons(context.Context, int) ([]TVDBSeason, bool, bool) {
	f.calls++
	return f.seasons, true, f.ok
}

func TestApplySeasonSource(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(ctx, "sqlite:"+filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := New(st, "")
	newDetail := func() *Detail {
		return &Detail{TVDBID: 74796, NumberOfSeasons: 2, Seasons: append([]Season(nil), bleachTMDB...)}
	}

	d := newDetail()
	s.applySeasonSource(ctx, Opts{}, d) // no source
	if d.SeasonNumbering != NumberingTMDB || len(d.Seasons) != 3 {
		t.Fatalf("no source: %+v", d)
	}

	src := &fakeSource{ok: false}
	s.SetSeasonSource(src)
	d = newDetail()
	s.applySeasonSource(ctx, Opts{}, d)
	if d.SeasonNumbering != NumberingTMDB || len(d.Seasons) != 3 || d.NumberOfSeasons != 2 || src.calls != 1 {
		t.Fatalf("ok=false: %+v", d)
	}

	src.ok, src.seasons = true, bleachTVDB(true)
	d = newDetail()
	s.applySeasonSource(ctx, Opts{Language: "fr-FR"}, d)
	if d.SeasonNumbering != NumberingTVDB || !d.SeasonNumberingDiffers || d.NumberOfSeasons != 17 || len(d.Seasons) != 18 {
		t.Fatalf("tvdb: numbering=%s differs=%v n=%d", d.SeasonNumbering, d.SeasonNumberingDiffers, d.NumberOfSeasons)
	}
	if d.Seasons[5].Name != "Saison 5" {
		t.Fatalf("localized name = %q", d.Seasons[5].Name)
	}
}
