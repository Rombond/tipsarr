// Package media serves TMDB metadata (discover, search, details), cached in the DB.
package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

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
func (s *Service) Discover(ctx context.Context, o Opts, mediaType string, genre, page int) (*List, error) {
	q := pageQ(o, page)
	q.Set("sort_by", "popularity.desc")
	if genre > 0 {
		q.Set("with_genres", strconv.Itoa(genre))
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

func (s *Service) Search(ctx context.Context, o Opts, query string, page int) (*SearchResult, error) {
	q := pageQ(o, page)
	q.Set("query", query)
	q.Set("include_adult", "false")
	var raw rawList
	if err := s.get(ctx, "/search/multi", q, ttlList, &raw); err != nil {
		return nil, err
	}
	out := &SearchResult{Page: raw.Page, TotalPages: raw.TotalPages, Items: []Item{}, People: []Person{}}
	for _, r := range raw.Results {
		switch r.MediaType {
		case "movie", "tv":
			out.Items = append(out.Items, r.toItem(""))
		case "person":
			out.People = append(out.People, Person{ID: r.ID, Name: r.Name, ProfilePath: r.ProfilePath, Department: r.Department})
		}
	}
	s.annotate(ctx, out.Items)
	return out, nil
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
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"belongs_to_collection"`
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
	q := url.Values{"language": {o.lang()}, "append_to_response": {"credits,recommendations,similar,external_ids"}}
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
		d.CollectionID, d.CollectionName = raw.BelongsToCollection.ID, raw.BelongsToCollection.Name
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
	s.annotate(ctx, d.Recommendations)
	s.annotate(ctx, d.Similar)
	one := []Item{d.Item}
	s.annotate(ctx, one)
	d.Availability, d.RequestStatus = one[0].Availability, one[0].RequestStatus
	if mediaType == "tv" {
		s.showAvailability(ctx, d)
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
