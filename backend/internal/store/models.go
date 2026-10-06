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
