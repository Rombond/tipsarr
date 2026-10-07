package ratings

import "testing"

func hit(title string, year int, vanity string, critics int) rtHit {
	h := rtHit{Title: title, ReleaseYear: year, Vanity: vanity}
	h.Rotten = &struct {
		AudienceScore int `json:"audienceScore"`
		CriticsScore  int `json:"criticsScore"`
	}{CriticsScore: critics, AudienceScore: 80}
	return h
}

// The search is by title, so the right film must beat the many that contain the name.
func TestBestHitPicksTheRightFilm(t *testing.T) {
	hits := []rtHit{
		hit("Blade Runner 2049", 2017, "blade_runner_2049", 88),
		hit("Runner", 2026, "runner_2026", 77),
		hit("The Runner", 2026, "the_runner_2026", 11),
		hit("The Maze Runner", 2014, "the_maze_runner", 66),
		hit("Runner Runner", 2013, "runner_runner", 8),
	}
	if h := bestHit(hits, "Runner", 2026); h == nil || h.Vanity != "runner_2026" {
		t.Fatalf("Runner 2026 -> %+v", h)
	}
	if h := bestHit(hits, "The Runner", 2026); h == nil || h.Vanity != "the_runner_2026" {
		t.Fatalf("The Runner 2026 -> %+v", h)
	}
	// the right title in the wrong decade must not win over nothing at all
	if h := bestHit(hits, "Runner", 1990); h != nil && h.Vanity == "runner_2026" {
		t.Fatalf("a 36-year gap should not match: %+v", h)
	}
	if h := bestHit(hits, "Something Completely Different", 2026); h != nil {
		t.Fatalf("unrelated title matched: %+v", h)
	}
}

func TestNormTitleAndSimilarity(t *testing.T) {
	if normTitle("Spider-Man: No Way Home!") != "spiderman no way home" {
		t.Fatal(normTitle("Spider-Man: No Way Home!"))
	}
	if similarity("up", "up") != 1 || similarity("up", "down") >= 0.25 {
		t.Fatal("similarity")
	}
}
