// Package boxoffice fetches the weekend box-office chart from Box Office Mojo's public pages,
// matches the titles to TMDB, and serves the stored history.
//
// The parser is written against the public HTML structure only (a table whose header row
// names the columns), so it keeps working if columns are added or reordered.
package boxoffice

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Entry struct {
	Pos            int
	Title          string
	WeekendGross   int64
	TotalGross     int64
	WeeksInRelease int
}

var (
	labelRe = regexp.MustCompile(`^[A-Z][a-z]+ \d{1,2}\s*-\s*(?:[A-Z][a-z]+ )?\d{1,2}, \d{4}$`)
	moneyRe = regexp.MustCompile(`[^\d]`)
)

var ErrNoChart = errors.New("no chart found in the page")

// ParseWeekend extracts the date label (e.g. "October 2-4, 2026") and the chart rows.
func ParseWeekend(page []byte) (label string, entries []Entry, err error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return "", nil, err
	}
	var header map[string]int
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode && label == "":
			if t := strings.TrimSpace(n.Data); labelRe.MatchString(t) {
				label = t
			}
		case n.Type == html.ElementNode && n.Data == "tr":
			if h := headerRow(n); h != nil {
				if _, ok := h["release"]; ok {
					header = h
				}
			} else if header != nil {
				if e, ok := dataRow(n, header); ok {
					entries = append(entries, e)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(entries) == 0 {
		return "", nil, ErrNoChart
	}
	return label, entries, nil
}

// headerRow returns column name (lower-cased) -> index when the row is made of <th> cells.
func headerRow(tr *html.Node) map[string]int {
	var cells []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "th" {
			cells = append(cells, c)
		}
	}
	if len(cells) == 0 {
		return nil
	}
	h := map[string]int{}
	for i, c := range cells {
		name := strings.ToLower(strings.TrimSpace(text(c)))
		if _, dup := h[name]; !dup {
			h[name] = i
		}
	}
	return h
}

func dataRow(tr *html.Node, h map[string]int) (Entry, bool) {
	var cells []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "td" {
			cells = append(cells, c)
		}
	}
	get := func(col string) *html.Node {
		if i, ok := h[col]; ok && i < len(cells) {
			return cells[i]
		}
		return nil
	}
	rank, rel := get("rank"), get("release")
	if rank == nil || rel == nil {
		return Entry{}, false
	}
	pos, err := strconv.Atoi(strings.TrimSpace(text(rank)))
	if err != nil {
		return Entry{}, false
	}
	title := ""
	if a := firstElement(rel, "a"); a != nil {
		title = strings.TrimSpace(text(a))
	} else {
		title = strings.TrimSpace(text(rel))
	}
	if title == "" {
		return Entry{}, false
	}
	e := Entry{Pos: pos, Title: title}
	if c := get("gross"); c != nil {
		e.WeekendGross = money(text(c))
	}
	if c := get("total gross"); c != nil {
		e.TotalGross = money(text(c))
	}
	if c := get("weeks"); c != nil {
		e.WeeksInRelease, _ = strconv.Atoi(strings.TrimSpace(text(c)))
	}
	return e, true
}

func money(s string) int64 {
	n, _ := strconv.ParseInt(moneyRe.ReplaceAllString(s, ""), 10, 64)
	return n
}

func text(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func firstElement(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == tag {
			return c
		}
		if f := firstElement(c, tag); f != nil {
			return f
		}
	}
	return nil
}
