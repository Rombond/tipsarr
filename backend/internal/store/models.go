package store

import "github.com/uptrace/bun"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID       string `bun:"id,pk" json:"id"` // Jellyfin user id
	Name     string `bun:"name" json:"name"`
	Role     string `bun:"role" json:"role"`
	Region   string `bun:"region" json:"region"`
	Language string `bun:"language" json:"language"`
	// RatingSource is the score shown on posters: tmdb, imdb, metacritic or rottenTomatoes ("" = tmdb)
	RatingSource string `bun:"rating_source" json:"ratingSource"`
	CreatedAt    int64  `bun:"created_at" json:"createdAt"`
	LastLoginAt  int64  `bun:"last_login_at" json:"lastLoginAt"`
}

type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	ID        string `bun:"id,pk"` // sha256 of the cookie token
	UserID    string `bun:"user_id"`
	UserAgent string `bun:"user_agent"`
	CreatedAt int64  `bun:"created_at"`
	ExpiresAt int64  `bun:"expires_at"`

	Platform   string `bun:"platform"` // web, ios or android
	DeviceName string `bun:"device_name"`
	AppVersion string `bun:"app_version"`
	LastSeenAt int64  `bun:"last_seen_at"`

	Renewed bool `bun:"-"` // set by auth when this request pushed ExpiresAt forward
}

// PublicID is what the API shows of a session: a prefix of its hash, enough to address it
// without ever exposing the full lookup key.
func (s *Session) PublicID() string { return s.ID[:16] }

type Setting struct {
	bun.BaseModel `bun:"table:settings"`

	Key   string `bun:"skey,pk"`
	Value string `bun:"svalue"`
}

type TMDBCache struct {
	bun.BaseModel `bun:"table:tmdb_cache"`

	Key       string `bun:"ckey,pk"`
	Body      string `bun:"body"`
	FetchedAt int64  `bun:"fetched_at"`
	ExpiresAt int64  `bun:"expires_at"`
}

type LibraryItem struct {
	bun.BaseModel `bun:"table:library_items"`

	MediaType  string `bun:"media_type,pk"`
	TMDBID     int64  `bun:"tmdb_id,pk"`
	JellyfinID string `bun:"jellyfin_id"`
	Title      string `bun:"title"`
	Year       int    `bun:"year"`
	RuntimeMin int    `bun:"runtime_min"`
	Rating10   int    `bun:"rating10"`  // community rating x 10
	AddedAt    int64  `bun:"added_at"`  // when Jellyfin first saw it (unix)
	Genres     string `bun:"genres"`    // |Action|Comedy|
	ImageTag   string `bun:"image_tag"` // Jellyfin's primary image tag (cache key for the poster)
}

type LibrarySeason struct {
	bun.BaseModel `bun:"table:library_seasons"`

	TMDBID       int64 `bun:"tmdb_id,pk"`
	SeasonNumber int   `bun:"season_number,pk"`
	EpisodeCount int   `bun:"episode_count"`
}

type WatchHistory struct {
	bun.BaseModel `bun:"table:watch_history"`

	UserID       string `bun:"user_id,pk"`
	MediaType    string `bun:"media_type,pk"`
	TMDBID       int64  `bun:"tmdb_id,pk"`
	LastPlayedAt int64  `bun:"last_played_at"`
	PlayCount    int    `bun:"play_count"`
}

// WatchEvent is one play session reported by Jellyfin's Playback Reporting plugin.
type WatchEvent struct {
	bun.BaseModel `bun:"table:watch_events"`

	SourceRowID int64  `bun:"source_rowid,pk"`
	UserID      string `bun:"user_id"`
	JellyfinID  string `bun:"jellyfin_id"`
	MediaType   string `bun:"media_type"` // movie | tv
	TMDBID      int64  `bun:"tmdb_id"`    // the movie, or the show of an episode; 0 = unknown
	Title       string `bun:"title"`
	PlayedAt    int64  `bun:"played_at"` // unix seconds
	Seconds     int    `bun:"seconds"`
}

// WatchTitle is the TMDB match of a played title that is no longer in the Jellyfin library.
type WatchTitle struct {
	bun.BaseModel `bun:"table:watch_titles"`

	MediaType string `bun:"media_type,pk"`
	Title     string `bun:"title,pk"`
	TMDBID    int64  `bun:"tmdb_id"` // 0 = no match
	Poster    string `bun:"poster"`  // TMDB poster path
	Year      int    `bun:"year"`
	Rating10  int    `bun:"rating10"`
	Genres    string `bun:"genres"` // |Action|Drama|
	CheckedAt int64  `bun:"checked_at"`
}

type UserHistoryState struct {
	bun.BaseModel `bun:"table:user_history_state"`

	UserID    string `bun:"user_id,pk"`
	Version   int64  `bun:"version"`
	Hash      string `bun:"hash"`
	UpdatedAt int64  `bun:"updated_at"`
}

type JobRun struct {
	bun.BaseModel `bun:"table:job_runs"`

	Name           string `bun:"name,pk" json:"name"`
	LastStartedAt  int64  `bun:"last_started_at" json:"lastStartedAt"`
	LastFinishedAt int64  `bun:"last_finished_at" json:"lastFinishedAt"`
	Status         string `bun:"status" json:"status"` // running | ok | error
	Message        string `bun:"message" json:"message"`
}

type ServarrInstance struct {
	bun.BaseModel `bun:"table:servarr_instances"`

	ID               string `bun:"id,pk" json:"id"`
	Kind             string `bun:"kind" json:"kind"`
	Name             string `bun:"name" json:"name"`
	URL              string `bun:"url" json:"url"`
	APIKey           string `bun:"api_key" json:"-"`
	QualityProfileID int    `bun:"quality_profile_id" json:"qualityProfileId"`
	RootFolder       string `bun:"root_folder" json:"rootFolder"`
	IsDefault        int    `bun:"is_default" json:"-"`
	CreatedAt        int64  `bun:"created_at" json:"-"`
	GenreRoots       string `bun:"genre_roots" json:"-"` // JSON {"<genre id>": "<root folder>"}
	AnimeRoot        string `bun:"anime_root" json:"-"`  // Sonarr: root folder (and series type) for anime
}

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusDeclined  = "declined"
	StatusFailed    = "failed"
	StatusAvailable = "available"
)

type Request struct {
	bun.BaseModel `bun:"table:requests"`

	ID            string `bun:"id,pk"`
	MediaType     string `bun:"media_type"`
	TMDBID        int64  `bun:"tmdb_id"`
	Title         string `bun:"title"`
	PosterPath    string `bun:"poster_path"`
	ReleaseDate   string `bun:"release_date"`
	RequestedBy   string `bun:"requested_by"`
	Status        string `bun:"status"`
	DecidedBy     string `bun:"decided_by"`
	DeclineReason string `bun:"decline_reason"`
	InstanceID    string `bun:"instance_id"`
	ServarrID     int64  `bun:"servarr_id"`
	SentAt        int64  `bun:"sent_at"`
	DryRun        int    `bun:"dry_run"`
	Error         string `bun:"error"`
	ProfileID     int    `bun:"profile_id"`  // chosen quality profile, 0 = instance default
	RootFolder    string `bun:"root_folder"` // chosen root folder, "" = instance default
	Source        string `bun:"source"`      // "" = a user asked; "radarr"/"sonarr" = imported from what they monitor
	CreatedAt     int64  `bun:"created_at"`
	UpdatedAt     int64  `bun:"updated_at"`
}

type RequestSeason struct {
	bun.BaseModel `bun:"table:request_seasons"`

	RequestID    string `bun:"request_id,pk"`
	SeasonNumber int    `bun:"season_number,pk"`
}

type Webhook struct {
	bun.BaseModel `bun:"table:webhooks"`

	ID        string `bun:"id,pk" json:"id"`
	Name      string `bun:"name" json:"name"`
	URL       string `bun:"url" json:"url"`
	Secret    string `bun:"secret" json:"-"`
	Events    string `bun:"events" json:"-"` // comma separated
	Enabled   int    `bun:"enabled" json:"-"`
	CreatedAt int64  `bun:"created_at" json:"-"`
}

type SuggestionRow struct {
	bun.BaseModel `bun:"table:suggestion_rows"`

	ID             string `bun:"id,pk"`
	UserID         string `bun:"user_id"`
	Kind           string `bun:"kind"` // account | because
	SeedType       string `bun:"seed_type"`
	SeedTMDBID     int64  `bun:"seed_tmdb_id"`
	SeedTitle      string `bun:"seed_title"`
	Position       int    `bun:"position"`
	GeneratedAt    int64  `bun:"generated_at"`
	HistoryVersion int64  `bun:"history_version"`
	Personal       int    `bun:"personal"` // 1 when built from the user's own history
}

type SuggestionItem struct {
	bun.BaseModel `bun:"table:suggestion_items"`

	RowID       string `bun:"row_id,pk"`
	Pos         int    `bun:"pos,pk"`
	MediaType   string `bun:"media_type"`
	TMDBID      int64  `bun:"tmdb_id"`
	Title       string `bun:"title"`
	PosterPath  string `bun:"poster_path"`
	ReleaseDate string `bun:"release_date"`
	VoteTenths  int    `bun:"vote_tenths"`
	Overview    string `bun:"overview"`
}

type BoxOfficeWeek struct {
	bun.BaseModel `bun:"table:boxoffice_weeks"`

	Region    string `bun:"region,pk" json:"region"`
	WeekKey   string `bun:"week_key,pk" json:"weekKey"`
	Label     string `bun:"label" json:"label"`
	FetchedAt int64  `bun:"fetched_at" json:"fetchedAt"`
}

type BoxOfficeEntry struct {
	bun.BaseModel `bun:"table:boxoffice_entries"`

	Region         string `bun:"region,pk"`
	WeekKey        string `bun:"week_key,pk"`
	Pos            int    `bun:"pos,pk"`
	Title          string `bun:"title"`
	WeekendGross   int64  `bun:"weekend_gross"`
	TotalGross     int64  `bun:"total_gross"`
	WeeksInRelease int    `bun:"weeks_in_release"`
	TMDBID         int64  `bun:"tmdb_id"`
	PosterPath     string `bun:"poster_path"`
	ReleaseDate    string `bun:"release_date"`
	VoteTenths     int    `bun:"vote_tenths"`
	Overview       string `bun:"overview"`
}

const (
	MarkWatchlist = "watchlist"
	MarkBlocklist = "blocklist"
)

// UserMark is a title a user put on their watchlist or blocklist (with a display snapshot).
type UserMark struct {
	bun.BaseModel `bun:"table:user_marks"`

	UserID      string `bun:"user_id,pk"`
	Kind        string `bun:"kind,pk"`
	MediaType   string `bun:"media_type,pk"`
	TMDBID      int64  `bun:"tmdb_id,pk"`
	Title       string `bun:"title"`
	PosterPath  string `bun:"poster_path"`
	ReleaseDate string `bun:"release_date"`
	VoteTenths  int    `bun:"vote_tenths"`
	CreatedAt   int64  `bun:"created_at"`
}

// BoxOfficeAlias pins a chart title (lower-cased) to a TMDB movie.
type BoxOfficeAlias struct {
	bun.BaseModel `bun:"table:boxoffice_aliases"`

	TitleKey  string `bun:"title_key,pk"`
	TMDBID    int64  `bun:"tmdb_id"`
	CreatedAt int64  `bun:"created_at"`
}
