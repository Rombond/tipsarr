package boxoffice

import (
	"os"
	"testing"
)

func TestParseWeekend(t *testing.T) {
	page, err := os.ReadFile("testdata/weekend.html")
	if err != nil {
		t.Fatal(err)
	}
	label, entries, err := ParseWeekend(page)
	if err != nil {
		t.Fatal(err)
	}
	if label != "October 2-4, 2026" {
		t.Fatalf("label = %q", label)
	}
	if len(entries) != 5 {
		t.Fatalf("entries = %d: %+v", len(entries), entries)
	}
	first := entries[0]
	if first.Pos != 1 || first.Title != "Verity" || first.WeekendGross != 32031011 || first.TotalGross != 32031011 || first.WeeksInRelease != 1 {
		t.Fatalf("first = %+v", first)
	}
	second := entries[1]
	if second.Title != "Resident Evil" || second.WeekendGross != 12618839 || second.TotalGross != 125013734 || second.WeeksInRelease != 3 {
		t.Fatalf("second = %+v", second)
	}
}

func TestParseRejectsPagesWithoutChart(t *testing.T) {
	if _, _, err := ParseWeekend([]byte("<html><body><p>nothing</p></body></html>")); err != ErrNoChart {
		t.Fatalf("err = %v", err)
	}
	// header present but no data rows
	if _, _, err := ParseWeekend([]byte("<table><tr><th>Rank</th><th>Release</th></tr></table>")); err != ErrNoChart {
		t.Fatalf("empty table err = %v", err)
	}
}

func TestMoney(t *testing.T) {
	for in, want := range map[string]int64{"$32,031,011": 32031011, "-": 0, "$0": 0, "": 0} {
		if got := money(in); got != want {
			t.Fatalf("money(%q) = %d", in, got)
		}
	}
}
