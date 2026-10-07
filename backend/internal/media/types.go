package media

// Availability / request state are filled in by later phases (library sync, requests).
// SettingDefaultLanguage is the app-wide default TMDB/interface language (e.g. "fr-FR"). A person's
// own language wins; empty means "follow the browser".
const SettingDefaultLanguage = "app.language"

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
	Page       int       `json:"page"`
	TotalPages int       `json:"totalPages"`
	Items      []Item    `json:"items"`
	People     []Person  `json:"people"`
	Keywords   []Keyword `json:"keywords" doc:"Tags matching the query (only when tags=true)"`
	Tagged     []Item    `json:"tagged" doc:"Titles carrying one of those tags that the text search did not return"`
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

type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CrewMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Job         string `json:"job" doc:"Director, Writer, Screenplay, Story, Novel, Editor, Producer, Composer, Cinematography, Creator"`
	ProfilePath string `json:"profilePath,omitempty"`
}

type Review struct {
	Author    string  `json:"author"`
	Rating    float64 `json:"rating,omitempty" doc:"The reviewer's own score out of 10, when given"`
	Content   string  `json:"content" doc:"Shortened to a few hundred characters"`
	URL       string  `json:"url,omitempty"`
	CreatedAt string  `json:"createdAt,omitempty"`
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
	Tagline                string       `json:"tagline,omitempty"`
	Status                 string       `json:"status,omitempty"`
	RuntimeMinutes         int          `json:"runtimeMinutes,omitempty"`
	Genres                 []Genre      `json:"genres"`
	OriginalLanguage       string       `json:"originalLanguage,omitempty"`
	Homepage               string       `json:"homepage,omitempty"`
	IMDbID                 string       `json:"imdbId,omitempty"`
	TVDBID                 int          `json:"tvdbId,omitempty"`
	Cast                   []CastMember `json:"cast"`
	Directors              []Person     `json:"directors"`
	Crew                   []CrewMember `json:"crew" doc:"Key crew (writers, editor, producers, composer...); creators for TV"`
	Reviews                []Review     `json:"reviews" doc:"A few TMDB user reviews"`
	Keywords               []Keyword    `json:"keywords" doc:"TMDB tags such as \"superhero\"; searchable"`
	VoteCount              int          `json:"voteCount,omitempty"`
	Languages              []string     `json:"languages,omitempty" doc:"Spoken languages, English names"`
	Countries              []string     `json:"countries,omitempty" doc:"Production countries"`
	NumberOfSeasons        int          `json:"numberOfSeasons,omitempty"`
	NumberOfEpisodes       int          `json:"numberOfEpisodes,omitempty"`
	Seasons                []Season     `json:"seasons,omitempty"`
	CollectionID           int          `json:"collectionId,omitempty"`
	CollectionName         string       `json:"collectionName,omitempty"`
	CollectionBackdropPath string       `json:"collectionBackdropPath,omitempty"`
	Studios                []string     `json:"studios,omitempty" doc:"Production companies (movies) or networks (TV)"`
	Budget                 int64        `json:"budget,omitempty"`
	Revenue                int64        `json:"revenue,omitempty"`
	TrailerKey             string       `json:"trailerKey,omitempty" doc:"YouTube video key of the official trailer"`
	WatchURL               string       `json:"watchUrl,omitempty" doc:"Deep link to the title in Jellyfin when it is in the library"`
	Recommendations        []Item       `json:"recommendations"`
	Similar                []Item       `json:"similar"`
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
