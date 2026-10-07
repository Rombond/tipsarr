package ratings

// Rotten Tomatoes scores for titles Radarr has none for, found the way Seerr does it: Rotten
// Tomatoes has no public API, but its own website searches an Algolia index with a key that is
// public in the site's code. The search is by title (there is no id lookup), so the best hit is
// chosen by scoring title and release year. The matching rules and the search parameters are
// adapted from Seerr (github.com/seerr-team/seerr, server/api/rating/rottentomatoes.ts, MIT licence).
// This is best effort: an unofficial source that can change or answer with the wrong film.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

const (
	rtEndpoint    = "https://79frdp12pn-dsn.algolia.net/1/indexes/*/queries"
	rtAgent       = "Algolia for JavaScript (4.14.3); Browser (lite)"
	rtAPIKey      = "175588f6e5f8319b27702e4cc4013561" // public: it is in rottentomatoes.com's own pages
	rtAppID       = "79FRDP12PN"
	rtInexact     = 0.25
	rtAlternate   = 0.8
	rtYearPenalty = 0.4
	rtMinimum     = 0.175
)

type rtHit struct {
	Title       string   `json:"title"`
	Titles      []string `json:"titles"`
	AKA         []string `json:"aka"`
	ReleaseYear int      `json:"releaseYear"`
	Vanity      string   `json:"vanity"`
	Rotten      *struct {
		AudienceScore int `json:"audienceScore"`
		CriticsScore  int `json:"criticsScore"`
	} `json:"rottenTomatoes"`
}

// RTScore is what Rotten Tomatoes says about a title.
type RTScore struct {
	Critics  int
	Audience int
	URL      string
}

var theWord = regexp.MustCompile(`(?i)\bthe\b ?`)

// rottenTomatoes looks a title up. kind is "movie" or "tv". (nil, nil) means no good match.
func rottenTomatoes(ctx context.Context, client *http.Client, userToken, kind, name string, year int) (*RTScore, error) {
	query := name
	if kind == "movie" {
		query = strings.TrimSpace(theWord.ReplaceAllString(name, ""))
	}
	body, _ := json.Marshal(map[string]any{"requests": []map[string]string{{
		"indexName": "content_rt", "query": query,
		"params": "filters=" + url.QueryEscape(`isEmsSearchable=1 AND type:"`+kind+`"`) + "&hitsPerPage=20",
	}}})
	q := url.Values{"x-algolia-agent": {rtAgent}, "x-algolia-api-key": {rtAPIKey}, "x-algolia-application-id": {rtAppID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rtEndpoint+"?"+q.Encode(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-algolia-usertoken", userToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("rotten tomatoes search unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("rotten tomatoes search refused")
	}
	var out struct {
		Results []struct {
			Index string  `json:"index"`
			Hits  []rtHit `json:"hits"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	for _, r := range out.Results {
		if r.Index != "content_rt" {
			continue
		}
		best := bestHit(r.Hits, name, year)
		if best == nil || best.Rotten == nil {
			return nil, nil
		}
		path := "/m/"
		if kind == "tv" {
			path = "/tv/"
		}
		return &RTScore{Critics: best.Rotten.CriticsScore, Audience: best.Rotten.AudienceScore, URL: "https://www.rottentomatoes.com" + path + best.Vanity}, nil
	}
	return nil, nil
}

// bestHit scores every hit as (title similarity) x (year closeness) x (half when it has no scores)
// and keeps the best one above a minimum.
func bestHit(hits []rtHit, name string, year int) *rtHit {
	var best *rtHit
	bestScore := rtMinimum
	for i := range hits {
		h := &hits[i]
		sc := titleScore(h, name) * yearScore(h, year)
		if h.Rotten == nil {
			sc *= 0.5
		}
		if sc > bestScore {
			best, bestScore = h, sc
		}
	}
	return best
}

func titleScore(h *rtHit, name string) float64 {
	want := normTitle(name)
	var best float64
	names := append(append([]string{h.Title}, h.AKA...), h.Titles...)
	for i, n := range names {
		s := similarity(normTitle(n), want)
		if i > 0 {
			s *= rtAlternate // alternate titles count less than the main one
		}
		best = max(best, s)
	}
	return best
}

// yearScore: same year 1.0, one off 0.6, two off 0.2, more 0; 1 when the year is unknown.
func yearScore(h *rtHit, year int) float64 {
	if year == 0 {
		return 1
	}
	d := h.ReleaseYear - year
	if d < 0 {
		d = -d
	}
	return max(0, 1-float64(d)*rtYearPenalty)
}

func similarity(a, b string) float64 {
	if a == b {
		return 1
	}
	return jaroWinkler(a, b) * rtInexact
}

// normTitle lower-cases and keeps letters, digits and spaces.
func normTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func jaroWinkler(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	window := max(len(ra), len(rb))/2 - 1
	if window < 0 {
		window = 0
	}
	ma, mb := make([]bool, len(ra)), make([]bool, len(rb))
	matches := 0
	for i := range ra {
		lo, hi := max(0, i-window), min(len(rb), i+window+1)
		for j := lo; j < hi; j++ {
			if !mb[j] && ra[i] == rb[j] {
				ma[i], mb[j] = true, true
				matches++
				break
			}
		}
	}
	if matches == 0 {
		return 0
	}
	trans, k := 0, 0
	for i := range ra {
		if !ma[i] {
			continue
		}
		for !mb[k] {
			k++
		}
		if ra[i] != rb[k] {
			trans++
		}
		k++
	}
	m := float64(matches)
	jaro := (m/float64(len(ra)) + m/float64(len(rb)) + (m-float64(trans)/2)/m) / 3
	prefix := 0
	for prefix < min(4, min(len(ra), len(rb))) && ra[prefix] == rb[prefix] {
		prefix++
	}
	if jaro < 0.7 {
		return jaro
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}
