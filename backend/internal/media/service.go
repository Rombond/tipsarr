// Package media serves TMDB metadata (discover, search, details), cached in the DB.
package media

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rombond/tipsarr/backend/internal/clients/tmdb"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

var (
	ErrNotConfigured = errors.New("TMDB API key is not configured")
	ErrNotFound      = tmdb.ErrNotFound
)

const (
	settingTMDBKey = "tmdb.api_key"
	ttlList        = time.Hour
	ttlDetail      = 24 * time.Hour
	defaultLang    = "en-US"
)

type Service struct {
	store   *store.Store
	baseURL string // TMDB base URL override (tests)
}

func New(s *store.Store, baseURL string) *Service { return &Service{store: s, baseURL: baseURL} }

// Opts carries per-user preferences.
type Opts struct {
	Language string // e.g. fr-FR; empty = en-US
	Region   string // e.g. FR; empty = none
}

func (o Opts) lang() string {
	if o.Language == "" {
		return defaultLang
	}
	return o.Language
}

// fetch returns the raw TMDB body for path+query, through the DB cache.
func (s *Service) fetch(ctx context.Context, path string, q url.Values, ttl time.Duration) ([]byte, error) {
	if q == nil {
		q = url.Values{}
	}
	key := path + "?" + q.Encode() // url.Values.Encode sorts keys; computed before the API key is added
	if len(key) > 191 {
		key = key[:150] + "#" + strconv.Itoa(hashString(key))
	}
	if body, ok, err := s.store.GetCache(ctx, key); err == nil && ok {
		return body, nil
	}
	apiKey, err := s.store.GetSetting(ctx, settingTMDBKey)
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, ErrNotConfigured
	}
	body, err := tmdb.New(s.baseURL, apiKey).Get(ctx, path, q)
	if err != nil {
		// TMDB down or unreachable: older data beats an error page (a 404 or a bad key is a real answer)
		if !errors.Is(err, ErrNotFound) && !errors.Is(err, tmdb.ErrInvalidKey) {
			if old, ok, serr := s.store.GetCacheStale(ctx, key); serr == nil && ok {
				slog.Warn("tmdb unavailable, serving cached copy", "path", path, "err", err)
				return old, nil
			}
		}
		return nil, err
	}
	_ = s.store.PutCache(ctx, key, body, ttl) // a cache write failure must not fail the request
	return body, nil
}

func hashString(s string) int {
	h := 5381
	for _, c := range s {
		h = h*33 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}

func (s *Service) get(ctx context.Context, path string, q url.Values, ttl time.Duration, out any) error {
	body, err := s.fetch(ctx, path, q, ttl)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// annotate fills Availability from the synced Jellyfin library. In lists a show that is in
// the library is reported "available"; the detail page refines that to "partial" using
// per-season episode counts. Library failures never break metadata requests.
func (s *Service) annotate(ctx context.Context, items []Item) {
	for _, mt := range []string{"movie", "tv"} {
		var ids []int
		for _, it := range items {
			if it.Type == mt {
				ids = append(ids, it.TMDBID)
			}
		}
		if len(ids) == 0 {
			continue
		}
		have, err := s.store.LibraryIDs(ctx, mt, ids)
		if err != nil {
			continue
		}
		active, _ := s.store.ActiveRequestStatuses(ctx, mt, ids)
		for i := range items {
			if items[i].Type != mt {
				continue
			}
			if have[items[i].TMDBID] {
				items[i].Availability = AvailabilityAvailable
			}
			items[i].RequestStatus = active[items[i].TMDBID]
		}
	}
}

// showAvailability compares TMDB season sizes with what the library holds.
func (s *Service) showAvailability(ctx context.Context, d *Detail) {
	have, err := s.store.LibraryIDs(ctx, "tv", []int{d.TMDBID})
	if err != nil || !have[d.TMDBID] {
		return
	}
	lib, err := s.store.LibrarySeasons(ctx, d.TMDBID)
	if err != nil {
		return
	}
	complete, anyEpisode := true, false
	for _, sn := range d.Seasons {
		if sn.Number == 0 || sn.EpisodeCount == 0 {
			continue // specials and announced-but-empty seasons don't count
		}
		if lib[sn.Number] > 0 {
			anyEpisode = true
		}
		if lib[sn.Number] < sn.EpisodeCount {
			complete = false
		}
	}
	switch {
	case complete && anyEpisode:
		d.Availability = AvailabilityAvailable
	default:
		d.Availability = AvailabilityPartial
	}
}

// ---- raw TMDB shapes ------------------------------------------------------

type rawItem struct {
	ID           int     `json:"id"`
	MediaType    string  `json:"media_type"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
	BackdropPath string  `json:"backdrop_path"`
	VoteAverage  float64 `json:"vote_average"`
	GenreIDs     []int   `json:"genre_ids"`
	// person results
	ProfilePath string `json:"profile_path"`
	Department  string `json:"known_for_department"`
}

// toItem converts a raw movie/tv result. fallbackType is used when media_type is absent.
func (r rawItem) toItem(fallbackType string) Item {
	t := r.MediaType
	if t == "" {
		t = fallbackType
	}
	title, date := r.Title, r.ReleaseDate
	if t == "tv" {
		title, date = r.Name, r.FirstAirDate
	}
	return Item{
		Type: t, TMDBID: r.ID, Title: title, ReleaseDate: date, Overview: r.Overview,
		PosterPath: r.PosterPath, BackdropPath: r.BackdropPath, VoteAverage: r.VoteAverage,
		GenreIDs: r.GenreIDs, Availability: AvailabilityNone,
	}
}

type rawList struct {
	Page       int       `json:"page"`
	TotalPages int       `json:"total_pages"`
	Results    []rawItem `json:"results"`
}

func (l rawList) toList(fallbackType string) List {
	out := List{Page: l.Page, TotalPages: l.TotalPages, Items: []Item{}}
	for _, r := range l.Results {
		if r.MediaType == "person" {
			continue
		}
		out.Items = append(out.Items, r.toItem(fallbackType))
	}
	return out
}

// ---- discover ----------------------------------------------------------------

func pageQ(o Opts, page int) url.Values {
	if page < 1 {
		page = 1
	}
	q := url.Values{"language": {o.lang()}, "page": {strconv.Itoa(page)}}
	if o.Region != "" {
		q.Set("region", o.Region)
	}
	return q
}

func (s *Service) list(ctx context.Context, path string, q url.Values, fallbackType string) (*List, error) {
	var raw rawList
	if err := s.get(ctx, path, q, ttlList, &raw); err != nil {
		return nil, err
	}
	l := raw.toList(fallbackType)
	s.annotate(ctx, l.Items)
	return &l, nil
}

func (s *Service) Trending(ctx context.Context, o Opts, page int) (*List, error) {
	return s.list(ctx, "/trending/all/week", pageQ(o, page), "")
}

// DiscoverMovies / DiscoverTV list by popularity, optionally filtered by genre id.
func (s *Service) Discover(ctx context.Context, o Opts, mediaType string, genre, keyword, page int) (*List, error) {
	q := pageQ(o, page)
	q.Set("sort_by", "popularity.desc")
	if genre > 0 {
		q.Set("with_genres", strconv.Itoa(genre))
	}
	if keyword > 0 {
		q.Set("with_keywords", strconv.Itoa(keyword))
	}
	return s.list(ctx, "/discover/"+mediaType, q, mediaType)
}

func (s *Service) Upcoming(ctx context.Context, o Opts, page int) (*List, error) {
	return s.list(ctx, "/movie/upcoming", pageQ(o, page), "movie")
}

func (s *Service) Genres(ctx context.Context, o Opts, mediaType string) ([]Genre, error) {
	var raw struct {
		Genres []Genre `json:"genres"`
	}
	if err := s.get(ctx, "/genre/"+mediaType+"/list", url.Values{"language": {o.lang()}}, 7*24*time.Hour, &raw); err != nil {
		return nil, err
	}
	if raw.Genres == nil {
		raw.Genres = []Genre{}
	}
	return raw.Genres, nil
}

// ---- search --------------------------------------------------------------------

func (s *Service) Search(ctx context.Context, o Opts, query string, page int, tags bool) (*SearchResult, error) {
	q := pageQ(o, page)
	q.Set("query", query)
	q.Set("include_adult", "false")
	var raw rawList
	if err := s.get(ctx, "/search/multi", q, ttlList, &raw); err != nil {
		return nil, err
	}
	out := &SearchResult{Page: raw.Page, TotalPages: raw.TotalPages, Items: []Item{}, People: []Person{}, Keywords: []Keyword{}, Tagged: []Item{}}
	for _, r := range raw.Results {
		switch r.MediaType {
		case "movie", "tv":
			out.Items = append(out.Items, r.toItem(""))
		case "person":
			out.People = append(out.People, Person{ID: r.ID, Name: r.Name, ProfilePath: r.ProfilePath, Department: r.Department})
		}
	}
	if tags && page <= 1 {
		s.tagged(ctx, o, query, out)
	}
	s.annotate(ctx, out.Items)
	return out, nil
}

// tagged adds the titles carrying a TMDB tag that matches the query ("shark" finds every
// movie/show tagged "shark"), minus the ones the text search already returned.
func (s *Service) tagged(ctx context.Context, o Opts, query string, out *SearchResult) {
	var kw struct {
		Results []Keyword `json:"results"`
	}
	if err := s.get(ctx, "/search/keyword", url.Values{"query": {query}}, ttlList, &kw); err != nil || len(kw.Results) == 0 {
		return
	}
	if len(kw.Results) > 3 {
		kw.Results = kw.Results[:3]
	}
	out.Keywords = kw.Results
	ids := make([]string, len(kw.Results))
	for i, k := range kw.Results {
		ids[i] = strconv.Itoa(k.ID)
	}
	have := map[string]bool{}
	for _, it := range out.Items {
		have[it.Type+strconv.Itoa(it.TMDBID)] = true
	}
	for _, mt := range []string{"movie", "tv"} {
		q := pageQ(o, 1)
		q.Set("sort_by", "popularity.desc")
		q.Set("with_keywords", strings.Join(ids, "|")) // any of the tags
		var raw rawList
		if err := s.get(ctx, "/discover/"+mt, q, ttlList, &raw); err != nil {
			continue
		}
		for _, r := range raw.Results {
			it := r.toItem(mt)
			if !have[it.Type+strconv.Itoa(it.TMDBID)] {
				have[it.Type+strconv.Itoa(it.TMDBID)] = true
				out.Tagged = append(out.Tagged, it)
			}
		}
	}
	s.annotate(ctx, out.Tagged)
}

// Keyword returns one tag by id (to title a tag page).
func (s *Service) Keyword(ctx context.Context, id int) (*Keyword, error) {
	var k Keyword
	if err := s.get(ctx, "/keyword/"+strconv.Itoa(id), nil, 7*24*time.Hour, &k); err != nil {
		return nil, err
	}
	return &k, nil
}

// ---- details ---------------------------------------------------------------------

type rawDetail struct {
	rawItem
	Tagline          string  `json:"tagline"`
	Status           string  `json:"status"`
	Runtime          int     `json:"runtime"`
	EpisodeRunTime   []int   `json:"episode_run_time"`
	Genres           []Genre `json:"genres"`
	OriginalLanguage string  `json:"original_language"`
	Homepage         string  `json:"homepage"`
	IMDbID           string  `json:"imdb_id"`
	ExternalIDs      struct {
		IMDbID string `json:"imdb_id"`
		TVDBID int    `json:"tvdb_id"`
	} `json:"external_ids"`
	NumberOfSeasons  int `json:"number_of_seasons"`
	NumberOfEpisodes int `json:"number_of_episodes"`
	Seasons          []struct {
		SeasonNumber int    `json:"season_number"`
		Name         string `json:"name"`
		EpisodeCount int    `json:"episode_count"`
		AirDate      string `json:"air_date"`
		PosterPath   string `json:"poster_path"`
	} `json:"seasons"`
	BelongsToCollection *struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		BackdropPath string `json:"backdrop_path"`
	} `json:"belongs_to_collection"`
	ProductionCompanies []struct {
		Name string `json:"name"`
	} `json:"production_companies"`
	Networks []struct {
		Name string `json:"name"`
	} `json:"networks"`
	Budget    int64 `json:"budget"`
	Revenue   int64 `json:"revenue"`
	VoteCount int   `json:"vote_count"`
	Keywords  struct {
		Keywords []Keyword `json:"keywords"` // movies
		Results  []Keyword `json:"results"`  // shows
	} `json:"keywords"`
	Reviews struct {
		Results []struct {
			Author        string `json:"author"`
			Content       string `json:"content"`
			URL           string `json:"url"`
			CreatedAt     string `json:"created_at"`
			AuthorDetails struct {
				Rating *float64 `json:"rating"`
			} `json:"author_details"`
		} `json:"results"`
	} `json:"reviews"`
	CreatedBy []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		ProfilePath string `json:"profile_path"`
	} `json:"created_by"`
	SpokenLanguages []struct {
		EnglishName string `json:"english_name"`
	} `json:"spoken_languages"`
	ProductionCountries []struct {
		Name string `json:"name"`
	} `json:"production_countries"`
	Videos struct {
		Results []struct {
			Key      string `json:"key"`
			Site     string `json:"site"`
			Type     string `json:"type"`
			Official bool   `json:"official"`
		} `json:"results"`
	} `json:"videos"`
	Credits struct {
		Cast []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
		} `json:"cast"`
		Crew []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Job         string `json:"job"`
			ProfilePath string `json:"profile_path"`
		} `json:"crew"`
	} `json:"credits"`
	Recommendations rawList `json:"recommendations"`
	Similar         rawList `json:"similar"`
}

func (s *Service) Detail(ctx context.Context, o Opts, mediaType string, id int) (*Detail, error) {
	q := url.Values{"language": {o.lang()}, "append_to_response": {"credits,recommendations,similar,external_ids,videos,keywords,reviews"}, "include_video_language": {"en,null"}}
	var raw rawDetail
	if err := s.get(ctx, "/"+mediaType+"/"+strconv.Itoa(id), q, ttlDetail, &raw); err != nil {
		return nil, err
	}
	d := &Detail{
		Item: raw.rawItem.toItem(mediaType), Tagline: raw.Tagline, Status: raw.Status,
		RuntimeMinutes: raw.Runtime, OriginalLanguage: raw.OriginalLanguage, Homepage: raw.Homepage,
		IMDbID: firstNonEmpty(raw.IMDbID, raw.ExternalIDs.IMDbID), TVDBID: raw.ExternalIDs.TVDBID,
		Genres: raw.Genres, NumberOfSeasons: raw.NumberOfSeasons, NumberOfEpisodes: raw.NumberOfEpisodes,
		Cast: []CastMember{}, Directors: []Person{},
		Recommendations: raw.Recommendations.toList(mediaType).Items,
		Similar:         raw.Similar.toList(mediaType).Items,
	}
	d.Type = mediaType
	if d.RuntimeMinutes == 0 && len(raw.EpisodeRunTime) > 0 {
		d.RuntimeMinutes = raw.EpisodeRunTime[0]
	}
	if d.Genres == nil {
		d.Genres = []Genre{}
	}
	for _, sn := range raw.Seasons {
		d.Seasons = append(d.Seasons, Season{Number: sn.SeasonNumber, Name: sn.Name, EpisodeCount: sn.EpisodeCount, AirDate: sn.AirDate, PosterPath: sn.PosterPath})
	}
	if raw.BelongsToCollection != nil {
		d.CollectionID, d.CollectionName, d.CollectionBackdropPath = raw.BelongsToCollection.ID, raw.BelongsToCollection.Name, raw.BelongsToCollection.BackdropPath
	}
	for i, c := range raw.Credits.Cast {
		if i >= 20 {
			break
		}
		d.Cast = append(d.Cast, CastMember{ID: c.ID, Name: c.Name, Character: c.Character, ProfilePath: c.ProfilePath})
	}
	for _, c := range raw.Credits.Crew {
		if c.Job == "Director" {
			d.Directors = append(d.Directors, Person{ID: c.ID, Name: c.Name, ProfilePath: c.ProfilePath, Department: "Directing"})
		}
	}
	d.Crew = keyCrew(raw, mediaType)
	d.Keywords = append(raw.Keywords.Keywords, raw.Keywords.Results...)
	if d.Keywords == nil {
		d.Keywords = []Keyword{}
	}
	d.Reviews = []Review{}
	for _, r := range raw.Reviews.Results {
		if len(d.Reviews) == 3 {
			break
		}
		rev := Review{Author: r.Author, Content: shorten(r.Content, 700), URL: r.URL, CreatedAt: r.CreatedAt}
		if r.AuthorDetails.Rating != nil {
			rev.Rating = *r.AuthorDetails.Rating
		}
		if rev.Content != "" {
			d.Reviews = append(d.Reviews, rev)
		}
	}
	d.VoteCount = raw.VoteCount
	for _, l := range raw.SpokenLanguages {
		d.Languages = append(d.Languages, l.EnglishName)
	}
	for _, c := range raw.ProductionCountries {
		d.Countries = append(d.Countries, c.Name)
	}
	if d.Overview == "" && !isEnglish(o.lang()) {
		d.Overview = s.englishOverview(ctx, mediaType, id)
	}
	d.TrailerKey = pickTrailer(raw)
	d.Budget, d.Revenue = raw.Budget, raw.Revenue
	if mediaType == "tv" {
		for _, n := range raw.Networks {
			d.Studios = append(d.Studios, n.Name)
		}
	} else {
		for _, c := range raw.ProductionCompanies {
			d.Studios = append(d.Studios, c.Name)
		}
	}
	s.annotate(ctx, d.Recommendations)
	s.annotate(ctx, d.Similar)
	one := []Item{d.Item}
	s.annotate(ctx, one)
	d.Availability, d.RequestStatus = one[0].Availability, one[0].RequestStatus
	if mediaType == "tv" {
		s.showAvailability(ctx, d)
	}
	if d.Availability != AvailabilityNone {
		d.WatchURL = s.watchURL(ctx, mediaType, d.TMDBID)
	}
	return d, nil
}

func (s *Service) Season(ctx context.Context, o Opts, tvID, season int) (*SeasonDetail, error) {
	var raw struct {
		SeasonNumber int    `json:"season_number"`
		Name         string `json:"name"`
		Overview     string `json:"overview"`
		AirDate      string `json:"air_date"`
		PosterPath   string `json:"poster_path"`
		Episodes     []struct {
			EpisodeNumber int     `json:"episode_number"`
			Name          string  `json:"name"`
			Overview      string  `json:"overview"`
			AirDate       string  `json:"air_date"`
			StillPath     string  `json:"still_path"`
			Runtime       int     `json:"runtime"`
			VoteAverage   float64 `json:"vote_average"`
		} `json:"episodes"`
	}
	path := "/tv/" + strconv.Itoa(tvID) + "/season/" + strconv.Itoa(season)
	if err := s.get(ctx, path, url.Values{"language": {o.lang()}}, ttlDetail, &raw); err != nil {
		return nil, err
	}
	out := &SeasonDetail{Number: raw.SeasonNumber, Name: raw.Name, Overview: raw.Overview, AirDate: raw.AirDate, PosterPath: raw.PosterPath, Episodes: []Episode{}}
	for _, e := range raw.Episodes {
		out.Episodes = append(out.Episodes, Episode{Number: e.EpisodeNumber, Name: e.Name, Overview: e.Overview, AirDate: e.AirDate, StillPath: e.StillPath, Runtime: e.Runtime, VoteAverage: e.VoteAverage})
	}
	return out, nil
}

func (s *Service) PersonDetail(ctx context.Context, o Opts, id int) (*PersonDetail, error) {
	var raw struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		ProfilePath  string `json:"profile_path"`
		Department   string `json:"known_for_department"`
		Biography    string `json:"biography"`
		Birthday     string `json:"birthday"`
		Deathday     string `json:"deathday"`
		PlaceOfBirth string `json:"place_of_birth"`
		Combined     struct {
			Cast []struct {
				rawItem
				Popularity float64 `json:"popularity"`
			} `json:"cast"`
		} `json:"combined_credits"`
	}
	q := url.Values{"language": {o.lang()}, "append_to_response": {"combined_credits"}}
	if err := s.get(ctx, "/person/"+strconv.Itoa(id), q, ttlDetail, &raw); err != nil {
		return nil, err
	}
	cast := raw.Combined.Cast
	sort.SliceStable(cast, func(i, j int) bool { return cast[i].Popularity > cast[j].Popularity })
	out := &PersonDetail{
		Person:    Person{ID: raw.ID, Name: raw.Name, ProfilePath: raw.ProfilePath, Department: raw.Department},
		Biography: raw.Biography, Birthday: raw.Birthday, Deathday: raw.Deathday, Birthplace: raw.PlaceOfBirth,
		Credits: []Item{},
	}
	seen := map[string]bool{}
	for _, c := range cast {
		if c.MediaType != "movie" && c.MediaType != "tv" {
			continue
		}
		key := c.MediaType + strconv.Itoa(c.ID)
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Credits = append(out.Credits, c.toItem(""))
		if len(out.Credits) >= 40 {
			break
		}
	}
	s.annotate(ctx, out.Credits)
	return out, nil
}

func (s *Service) Collection(ctx context.Context, o Opts, id int) (*CollectionDetail, error) {
	var raw struct {
		ID           int       `json:"id"`
		Name         string    `json:"name"`
		Overview     string    `json:"overview"`
		PosterPath   string    `json:"poster_path"`
		BackdropPath string    `json:"backdrop_path"`
		Parts        []rawItem `json:"parts"`
	}
	if err := s.get(ctx, "/collection/"+strconv.Itoa(id), url.Values{"language": {o.lang()}}, ttlDetail, &raw); err != nil {
		return nil, err
	}
	out := &CollectionDetail{ID: raw.ID, Name: raw.Name, Overview: raw.Overview, PosterPath: raw.PosterPath, BackdropPath: raw.BackdropPath, Parts: []Item{}}
	for _, p := range raw.Parts {
		out.Parts = append(out.Parts, p.toItem("movie"))
	}
	sort.SliceStable(out.Parts, func(i, j int) bool { return out.Parts[i].ReleaseDate < out.Parts[j].ReleaseDate })
	s.annotate(ctx, out.Parts)
	return out, nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ---- helpers for suggestions and box office ---------------------------------------------

// Annotate fills availability and request status (exported for other services).
func (s *Service) Annotate(ctx context.Context, items []Item) { s.annotate(ctx, items) }

// Related returns TMDB's recommendations and similar titles for a title, unannotated and
// cached. Either list may be empty.
func (s *Service) Related(ctx context.Context, o Opts, mediaType string, id int) (recs, similar []Item, err error) {
	q := url.Values{"language": {o.lang()}, "page": {"1"}}
	base := "/" + mediaType + "/" + strconv.Itoa(id)
	var r, sm rawList
	if err = s.get(ctx, base+"/recommendations", q, ttlDetail, &r); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, nil, err
	}
	if err = s.get(ctx, base+"/similar", q, ttlDetail, &sm); err != nil && !errors.Is(err, ErrNotFound) {
		return nil, nil, err
	}
	return r.toList(mediaType).Items, sm.toList(mediaType).Items, nil
}

// SearchMovies searches movies by title (cached, unannotated).
func (s *Service) SearchMovies(ctx context.Context, o Opts, title string) ([]Item, error) {
	q := url.Values{"language": {o.lang()}, "query": {title}, "include_adult": {"false"}, "page": {"1"}}
	var raw rawList
	if err := s.get(ctx, "/search/movie", q, ttlList, &raw); err != nil {
		return nil, err
	}
	return raw.toList("movie").Items, nil
}

// SearchTitle searches movies or shows ("movie" / "tv") by title (cached, unannotated).
func (s *Service) SearchTitle(ctx context.Context, mediaType, language, title string) ([]Item, error) {
	if mediaType != "tv" {
		return s.SearchMovies(ctx, Opts{Language: language}, title)
	}
	q := url.Values{"language": {Opts{Language: language}.lang()}, "query": {title}, "include_adult": {"false"}, "page": {"1"}}
	var raw rawList
	if err := s.get(ctx, "/search/tv", q, ttlList, &raw); err != nil {
		return nil, err
	}
	return raw.toList("tv").Items, nil
}

// TrendingItems returns trending titles without availability annotation.
func (s *Service) TrendingItems(ctx context.Context, o Opts, page int) ([]Item, error) {
	var raw rawList
	if err := s.get(ctx, "/trending/all/week", pageQ(o, page), ttlList, &raw); err != nil {
		return nil, err
	}
	return raw.toList("").Items, nil
}

// pickTrailer returns the YouTube key of the best trailer: official first, then any trailer.
func pickTrailer(raw rawDetail) string {
	fallback := ""
	for _, v := range raw.Videos.Results {
		if v.Site != "YouTube" || v.Type != "Trailer" {
			continue
		}
		if v.Official {
			return v.Key
		}
		if fallback == "" {
			fallback = v.Key
		}
	}
	return fallback
}

// Settings read by watchURL.
const (
	settingJellyfinURL       = "jellyfin.url"
	SettingJellyfinPublicURL = "jellyfin.public_url"
)

// watchURL builds a deep link into Jellyfin's web UI for a title in the library ("" if unknown).
// The public URL (what browsers can reach) falls back to the URL Tipsarr itself uses.
func (s *Service) watchURL(ctx context.Context, mediaType string, tmdbID int) string {
	id := s.store.LibraryJellyfinID(ctx, mediaType, int64(tmdbID))
	if id == "" {
		return ""
	}
	base, _ := s.store.GetSetting(ctx, SettingJellyfinPublicURL)
	if base == "" {
		base, _ = s.store.GetSetting(ctx, settingJellyfinURL)
	}
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/web/#/details?id=" + id
}

// keyCrew picks the crew worth showing: writers, editor, producers, composer, cinematographer
// (the director has its own field) and, for shows, the creators.
func keyCrew(raw rawDetail, mediaType string) []CrewMember {
	out := []CrewMember{}
	seen := map[string]bool{}
	add := func(id int, name, job, profile string) {
		k := strconv.Itoa(id) + "/" + job
		if id == 0 || seen[k] {
			return
		}
		seen[k] = true
		out = append(out, CrewMember{ID: id, Name: name, Job: job, ProfilePath: profile})
	}
	if mediaType == "tv" {
		for _, c := range raw.CreatedBy {
			add(c.ID, c.Name, "Creator", c.ProfilePath)
		}
	}
	// TMDB job -> the label we expose (several jobs fold into "Writer")
	label := map[string]string{
		"Writer": "Writer", "Screenplay": "Writer", "Story": "Writer", "Novel": "Writer", "Characters": "Writer",
		"Editor": "Editor", "Producer": "Producer", "Original Music Composer": "Composer", "Director of Photography": "Cinematography",
	}
	limit := map[string]int{"Writer": 4, "Editor": 2, "Producer": 3, "Composer": 2, "Cinematography": 2}
	count := map[string]int{}
	for _, c := range raw.Credits.Crew {
		l, ok := label[c.Job]
		if !ok || count[l] >= limit[l] {
			continue
		}
		before := len(out)
		add(c.ID, c.Name, l, c.ProfilePath)
		if len(out) > before {
			count[l]++
		}
	}
	return out
}

// shorten cuts review text at a word boundary and strips markdown-ish line noise.
func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(strings.ReplaceAll(s, "\r", " ")), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	cut := []rune(s)[:n]
	if i := strings.LastIndex(string(cut), " "); i > n/2 {
		return strings.TrimRight(string(cut)[:i], " ,.;:-") + "…"
	}
	return string(cut) + "…"
}

func isEnglish(lang string) bool { return lang == "" || strings.HasPrefix(strings.ToLower(lang), "en") }

type basicInfo struct {
	Title      string `json:"title"`
	Name       string `json:"name"`
	Overview   string `json:"overview"`
	PosterPath string `json:"poster_path"`
}

func (s *Service) basic(ctx context.Context, mediaType string, id int, lang string) (basicInfo, error) {
	var b basicInfo
	err := s.get(ctx, "/"+mediaType+"/"+strconv.Itoa(id), url.Values{"language": {lang}}, ttlDetail, &b)
	return b, err
}

// englishOverview is the fallback for titles TMDB has no translation for: the English text is
// better than an empty box.
func (s *Service) englishOverview(ctx context.Context, mediaType string, id int) string {
	b, err := s.basic(ctx, mediaType, id, defaultLang)
	if err != nil {
		return ""
	}
	return b.Overview
}

// Localize rewrites title, overview and poster of items in the person's language. It is for
// snapshots stored in English (the box-office charts); results come from TMDB's cached basic
// details, a few at a time. Anything that fails keeps its snapshot values.
func (s *Service) Localize(ctx context.Context, o Opts, items []Item) {
	lang := o.lang()
	if isEnglish(lang) || len(items) == 0 {
		return
	}
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(it *Item) {
			defer wg.Done()
			defer func() { <-sem }()
			b, err := s.basic(ctx, it.Type, it.TMDBID, lang)
			if err != nil {
				return
			}
			if t := firstNonEmpty(b.Title, b.Name); t != "" {
				it.Title = t
			}
			if b.PosterPath != "" {
				it.PosterPath = b.PosterPath
			}
			it.Overview = b.Overview
			if it.Overview == "" {
				it.Overview = s.englishOverview(ctx, it.Type, it.TMDBID)
			}
		}(&items[i])
	}
	wg.Wait()
}

// PosterRef identifies a title for Posters.
type PosterRef struct {
	Type   string // movie or tv
	TMDBID int
}

// Posters returns the TMDB poster path of each title in the person's language ("" when TMDB has none or
// the call failed). It reads TMDB's cached basic details, a few at a time, for lists whose own posters
// come from somewhere else (Jellyfin's library images, snapshots stored when a request was made).
func (s *Service) Posters(ctx context.Context, o Opts, refs []PosterRef) []string {
	out := make([]string, len(refs))
	lang := o.lang()
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for i, ref := range refs {
		if ref.TMDBID <= 0 || (ref.Type != "movie" && ref.Type != "tv") {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, ref PosterRef) {
			defer wg.Done()
			defer func() { <-sem }()
			if b, err := s.basic(ctx, ref.Type, ref.TMDBID, lang); err == nil {
				out[i] = b.PosterPath
			}
		}(i, ref)
	}
	wg.Wait()
	return out
}
