package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/avatars"
	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type profileStats struct {
	Requests  int `json:"requests"`
	Movies    int `json:"movies" doc:"Requested movies"`
	Shows     int `json:"shows" doc:"Requested shows"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Available int `json:"available"`
	Declined  int `json:"declined"`
	Failed    int `json:"failed"`
	Watchlist int `json:"watchlist"`
	Watched   int `json:"watched" doc:"Titles watched according to the Jellyfin history"`
}

type profileView struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Role         string       `json:"role" enum:"admin,user"`
	Region       string       `json:"region"`
	Language     string       `json:"language"`
	RatingSource string       `json:"ratingSource"`
	CreatedAt    int64        `json:"createdAt"`
	LastLoginAt  int64        `json:"lastLoginAt"`
	Stats        profileStats `json:"stats"`
	HasUpload    bool         `json:"hasUploadedAvatar" doc:"The person uploaded their own picture"`
}

func registerProfile(api huma.API, d Deps) {
	huma.Register(api, huma.Operation{
		OperationID: "userProfile", Method: http.MethodGet, Path: "/users/{id}",
		Summary:     "A user's profile and request statistics",
		Description: "Everyone can read their own profile; admins can read anyone's.",
		Tags:        []string{"users"}, Security: []map[string][]string{{"session": {}}, {"bearer": {}}},
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound},
	}, func(ctx context.Context, in *struct {
		ID string `path:"id" maxLength:"64"`
	}) (*struct{ Body profileView }, error) {
		me, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if me.ID != in.ID && me.Role != store.RoleAdmin {
			return nil, fail(403, "forbidden", "you can only open your own profile")
		}
		u, err := d.Store.GetUser(ctx, in.ID)
		if err != nil {
			return nil, fail(404, "not_found", "not found")
		}
		rs, err := d.Store.UserRequestStats(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		marks, err := d.Store.Marks(ctx, u.ID, "watchlist")
		if err != nil {
			return nil, err
		}
		watched, err := d.Store.WatchedCount(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		out := profileView{ID: u.ID, Name: u.Name, Role: u.Role, Region: u.Region, Language: u.Language, RatingSource: u.RatingSource, CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt,
			Stats: profileStats{Requests: rs.Total, Movies: rs.Movies, Shows: rs.Shows, Pending: rs.Pending, Approved: rs.Approved,
				Available: rs.Available, Declined: rs.Declined, Failed: rs.Failed, Watchlist: len(marks), Watched: watched}}
		out.HasUpload = d.Avatars != nil && d.Avatars.HasUpload(u.ID)
		return &struct{ Body profileView }{out}, nil
	})
}

var avatarIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// avatarHandler proxies a user's Jellyfin profile picture (404 when they have none), so the
// browser only ever talks to Tipsarr.
func avatarHandler(d Deps) http.HandlerFunc {
	client := &http.Client{Timeout: 10 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r.Context()) == nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		id := chiParam(r, "id")
		// a picture uploaded in Tipsarr wins, then the LDAP one, then Jellyfin's
		if d.Avatars != nil {
			name := ""
			if u, err := d.Store.GetUser(r.Context(), id); err == nil {
				name = u.Name
			}
			if path := d.Avatars.Path(r.Context(), id, name); path != "" {
				w.Header().Set("Cache-Control", "private, max-age=300")
				http.ServeFile(w, r, path)
				return
			}
		}
		base, err := d.Store.GetSetting(r.Context(), "jellyfin.url")
		if !avatarIDRe.MatchString(id) || err != nil || base == "" {
			http.NotFound(w, r)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, strings.TrimRight(base, "/")+"/Users/"+url.PathEscape(id)+"/Images/Primary?maxWidth=256", nil)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "avatar fetch failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		ct := resp.Header.Get("Content-Type")
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(ct, "image/") {
			w.Header().Set("Cache-Control", "private, max-age=300")
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "private, max-age=3600")
		_, _ = io.Copy(w, io.LimitReader(resp.Body, 5<<20))
	}
}

// avatarUploadHandler stores the picture sent in the request body for the signed-in user.
func avatarUploadHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r.Context())
		if u == nil || d.Avatars == nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodDelete:
			if err := d.Avatars.Delete(u.ID); err != nil {
				http.Error(w, "could not remove the picture", http.StatusInternalServerError)
				return
			}
		default:
			err := d.Avatars.Save(u.ID, http.MaxBytesReader(w, r.Body, 5<<20))
			switch {
			case errors.Is(err, avatars.ErrTooLarge):
				http.Error(w, "picture_too_large", http.StatusRequestEntityTooLarge)
				return
			case errors.Is(err, avatars.ErrNotImage):
				http.Error(w, "picture_invalid", http.StatusUnprocessableEntity)
				return
			case err != nil:
				http.Error(w, "could not save the picture", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
