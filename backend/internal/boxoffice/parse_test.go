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

func TestLabelDates(t *testing.T) {
	for label, want := range map[string][2]string{
		"October 2-4, 2026":            {"2026-10-02", "2026-10-04"},
		"September 30-October 4, 2026": {"2026-09-30", "2026-10-04"},
		"December 29-January 4, 2027":  {"2026-12-29", "2027-01-04"},
		"not a label":                  {"", ""},
	} {
		s, e := LabelDates(label)
		if s != want[0] || e != want[1] {
			t.Errorf("%q: got %s..%s, want %v", label, s, e, want)
		}
	}
}
