package requests

import (
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/clients/servarr"
)

func TestShowProgress(t *testing.T) {
	var sr servarr.Series
	sr.Statistics.EpisodeCount, sr.Statistics.EpisodeFileCount = 20, 10
	add := func(n, files int) {
		sr.Seasons = append(sr.Seasons, struct {
			SeasonNumber int  `json:"seasonNumber"`
			Monitored    bool `json:"monitored"`
			Statistics   struct {
				EpisodeFileCount int `json:"episodeFileCount"`
				EpisodeCount     int `json:"episodeCount"`
			} `json:"statistics"`
		}{SeasonNumber: n, Monitored: true})
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
