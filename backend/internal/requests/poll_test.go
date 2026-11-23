package requests

import (
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
)

func TestShowProgress(t *testing.T) {
	var sr servarr.Series
	sr.Statistics.EpisodeCount, sr.Statistics.EpisodeFileCount = 20, 10
	add := func(n, files int) {
		sr.Seasons = append(sr.Seasons, servarr.SeriesSeason{SeasonNumber: n, Monitored: true})
		last := &sr.Seasons[len(sr.Seasons)-1]
		last.Statistics.EpisodeCount, last.Statistics.EpisodeFileCount = 10, files
	}
	add(1, 10) // complete
	add(2, 0)  // two episodes half-downloaded through a season pack
	recs := []servarr.QueueItem{
		{Season: 2, Size: 1000, SizeLeft: 500, Status: "downloading"},
		{Season: 2, Size: 1000, SizeLeft: 500, Status: "downloading"},
	}
	total, seasons := showProgress(&sr, recs)
	if len(seasons) != 2 || seasons[0].Percent != 100 || seasons[1].Percent != 10 {
		t.Fatalf("seasons = %+v", seasons)
	}
	if total != 55 { // (10 files + 2 x 0.5) / 20
		t.Fatalf("total = %d", total)
	}
	if p := combine(recs); p.Percent != 50 {
		t.Fatalf("combine = %+v", p)
	}
}

// An airing season (issue #50): Sonarr's episodeCount only counts the aired episodes, so 3 of 3 on
// disk must still read 3/8 and keep the request open.
func TestShowProgressAiring(t *testing.T) {
	var sr servarr.Series
	sr.Statistics.EpisodeCount, sr.Statistics.EpisodeFileCount = 3, 3
	sr.Seasons = []servarr.SeriesSeason{{SeasonNumber: 1, Monitored: true}}
	st := &sr.Seasons[0].Statistics
	st.EpisodeCount, st.EpisodeFileCount, st.TotalEpisodeCount, st.NextAiring = 3, 3, 8, "2026-11-27T05:00:00Z"

	total, seasons := showProgress(&sr, nil)
	if len(seasons) != 1 || seasons[0].Percent != 37 || seasons[0].Episodes != 3 || seasons[0].Total != 8 || !seasons[0].Airing {
		t.Fatalf("seasons = %+v", seasons)
	}
	if total != 37 {
		t.Fatalf("total = %d", total)
	}
	if sr.Complete() || sr.SeasonsDone([]int{1}) || sr.SeasonsDone(nil) {
		t.Fatal("an airing season counts as done")
	}

	// the last episode aired and is on disk: done
	st.EpisodeCount, st.EpisodeFileCount, st.NextAiring = 8, 8, ""
	sr.Statistics.EpisodeCount, sr.Statistics.EpisodeFileCount = 8, 8
	if !sr.Complete() || !sr.SeasonsDone([]int{1}) {
		t.Fatal("a finished season is not done")
	}
	if _, seasons := showProgress(&sr, nil); seasons[0].Percent != 100 || seasons[0].Airing {
		t.Fatalf("seasons = %+v", seasons)
	}
}

// Only the requested seasons decide: a later season airing (monitored by Sonarr's "new seasons")
// does not hold back a request for a finished one; an unknown season falls back to Complete.
func TestSeasonsDoneRequestedOnly(t *testing.T) {
	var sr servarr.Series
	sr.Statistics.EpisodeCount, sr.Statistics.EpisodeFileCount = 12, 11
	sr.Seasons = []servarr.SeriesSeason{{SeasonNumber: 1, Monitored: true}, {SeasonNumber: 2, Monitored: true}}
	s1, s2 := &sr.Seasons[0].Statistics, &sr.Seasons[1].Statistics
	s1.EpisodeCount, s1.EpisodeFileCount, s1.TotalEpisodeCount = 10, 10, 10
	s2.EpisodeCount, s2.EpisodeFileCount, s2.TotalEpisodeCount, s2.NextAiring = 2, 1, 8, "2026-12-01T00:00:00Z"
	if !sr.SeasonsDone([]int{1}) {
		t.Fatal("finished season 1 held back by airing season 2")
	}
	if sr.SeasonsDone([]int{1, 2}) || sr.SeasonsDone([]int{7}) {
		t.Fatal("airing season 2 or unknown season counted as done")
	}
}
