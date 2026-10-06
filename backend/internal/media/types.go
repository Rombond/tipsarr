package media

// Availability / request state are filled in by later phases (library sync, requests).
const (
	AvailabilityNone      = "none"
	AvailabilityPartial   = "partial"
	AvailabilityAvailable = "available"
)

type Item struct {
	Type          string  `json:"type" enum:"movie,tv" doc:"movie or tv"`
	TMDBID        int     `json:"tmdbId"`
	Title         string  `json:"title"`
	ReleaseDate   string  `json:"releaseDate,omitempty"`
	Overview      string  `json:"overview,omitempty"`
	PosterPath    string  `json:"posterPath,omitempty" doc:"TMDB path, load via /images/tmdb/{size}/{file}"`
	BackdropPath  string  `json:"backdropPath,omitempty"`
	VoteAverage   float64 `json:"voteAverage"`
	GenreIDs      []int   `json:"genreIds,omitempty"`
	Availability  string  `json:"availability" enum:"none,partial,available" doc:"Library availability (filled by library sync)"`
	RequestStatus string  `json:"requestStatus,omitempty" enum:"pending,approved" doc:"Set while an active request exists for this title"`
}

type Person struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProfilePath string `json:"profilePath,omitempty"`
	Department  string `json:"department,omitempty"`
}

type List struct {
	Page       int    `json:"page"`
	TotalPages int    `json:"totalPages"`
	Items      []Item `json:"items"`
}

type SearchResult struct {
	Page       int      `json:"page"`
	TotalPages int      `json:"totalPages"`
	Items      []Item   `json:"items"`
	People     []Person `json:"people"`
}

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character,omitempty"`
	ProfilePath string `json:"profilePath,omitempty"`
}

type Season struct {
	Number       int    `json:"number"`
	Name         string `json:"name"`
	EpisodeCount int    `json:"episodeCount"`
	AirDate      string `json:"airDate,omitempty"`
	PosterPath   string `json:"posterPath,omitempty"`
}

type Detail struct {
	Item
	Tagline          string       `json:"tagline,omitempty"`
	Status           string       `json:"status,omitempty"`
	RuntimeMinutes   int          `json:"runtimeMinutes,omitempty"`
	Genres           []Genre      `json:"genres"`
	OriginalLanguage string       `json:"originalLanguage,omitempty"`
	Homepage         string       `json:"homepage,omitempty"`
	IMDbID           string       `json:"imdbId,omitempty"`
	TVDBID           int          `json:"tvdbId,omitempty"`
	Cast             []CastMember `json:"cast"`
	Directors        []Person     `json:"directors"`
	NumberOfSeasons  int          `json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes int          `json:"numberOfEpisodes,omitempty"`
	Seasons          []Season     `json:"seasons,omitempty"`
	CollectionID     int          `json:"collectionId,omitempty"`
	CollectionName   string       `json:"collectionName,omitempty"`
	Recommendations  []Item       `json:"recommendations"`
	Similar          []Item       `json:"similar"`
}

type Episode struct {
	Number      int     `json:"number"`
	Name        string  `json:"name"`
	Overview    string  `json:"overview,omitempty"`
	AirDate     string  `json:"airDate,omitempty"`
	StillPath   string  `json:"stillPath,omitempty"`
	Runtime     int     `json:"runtimeMinutes,omitempty"`
	VoteAverage float64 `json:"voteAverage"`
}

type SeasonDetail struct {
	Number     int       `json:"number"`
	Name       string    `json:"name"`
	Overview   string    `json:"overview,omitempty"`
	AirDate    string    `json:"airDate,omitempty"`
	PosterPath string    `json:"posterPath,omitempty"`
	Episodes   []Episode `json:"episodes"`
}

type PersonDetail struct {
	Person
	Biography  string `json:"biography,omitempty"`
	Birthday   string `json:"birthday,omitempty"`
	Deathday   string `json:"deathday,omitempty"`
	Birthplace string `json:"birthplace,omitempty"`
	Credits    []Item `json:"credits" doc:"Cast credits, most popular first"`
}

type CollectionDetail struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Overview     string `json:"overview,omitempty"`
	PosterPath   string `json:"posterPath,omitempty"`
	BackdropPath string `json:"backdropPath,omitempty"`
	Parts        []Item `json:"parts"`
}
