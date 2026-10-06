package store

import "github.com/uptrace/bun"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID          string `bun:"id,pk" json:"id"` // Jellyfin user id
	Name        string `bun:"name" json:"name"`
	Role        string `bun:"role" json:"role"`
	Region      string `bun:"region" json:"region"`
	Language    string `bun:"language" json:"language"`
	CreatedAt   int64  `bun:"created_at" json:"createdAt"`
	LastLoginAt int64  `bun:"last_login_at" json:"lastLoginAt"`
}

type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	ID        string `bun:"id,pk"` // sha256 of the cookie token
	UserID    string `bun:"user_id"`
	UserAgent string `bun:"user_agent"`
	CreatedAt int64  `bun:"created_at"`
	ExpiresAt int64  `bun:"expires_at"`
}

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
