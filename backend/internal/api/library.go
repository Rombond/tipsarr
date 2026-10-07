package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rombond/tipsarr/backend/internal/store"
	"github.com/danielgtaylor/huma/v2"
)

type libraryItem struct {
	Type       string   `json:"type" enum:"movie,tv"`
	TMDBID     int64    `json:"tmdbId"`
	Title      string   `json:"title"`
	Year       int      `json:"year,omitempty"`
	RuntimeMin int      `json:"runtimeMinutes,omitempty" doc:"Movie length, or typical episode length for a show"`
	Rating     float64  `json:"rating,omitempty" doc:"Jellyfin community rating out of 10"`
	Genres     []string `json:"genres"`
	AddedAt    int64    `json:"addedAt" doc:"When Jellyfin added it (unix seconds)"`
	PosterURL  string   `json:"posterUrl,omitempty" doc:"Poster served from Jellyfin through Tipsarr"`
	Plays      int      `json:"plays" doc:"Total plays by everyone"`
	Viewers    int      `json:"viewers" doc:"People who watched it"`
	Watched    bool     `json:"watched" doc:"The logged-in user watched it"`
}

type libraryList struct {
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	TotalPages int           `json:"totalPages"`
	Items      []libraryItem `json:"items"`
}

type libraryFacets struct {
	Movies     int          `json:"movies"`
	Shows      int          `json:"shows"`
	Genres     []genreCount `json:"genres" doc:"Most common first"`
	YearMin    int          `json:"yearMin"`
	YearMax    int          `json:"yearMax"`
	MaxRuntime int          `json:"maxRuntimeMinutes"`
}

type genreCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var (
	jfIDRe  = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)
	jfTagRe = regexp.MustCompile(`^[A-Za-z0-9]{0,64}$`)
)

func registerLibrary(api huma.API, d Deps) {
	sec := []map[string][]string{{"session": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "listLibrary", Method: http.MethodGet, Path: "/library",
		Summary:     "Browse what is in Jellyfin, with filters and sorting",
		Description: "Data comes from the library sync (genres, year, rating, runtime, date added from Jellyfin). `genre` may be repeated: a title must have all of them. `watched` is about the logged-in user.",
		Tags:        []string{"library"}, Security: sec, Errors: []int{http.StatusUnauthorized, http.StatusForbidden},
	}, func(ctx context.Context, in *struct {
		Type       string   `query:"type" enum:"all,movie,tv" default:"all"`
		Q          string   `query:"q" maxLength:"100" doc:"Part of the title"`
		Genre      []string `query:"genre,explode" maxItems:"6" doc:"Repeat for several; all must match"`
		YearFrom   int      `query:"yearFrom" minimum:"0" maximum:"2200"`
		YearTo     int      `query:"yearTo" minimum:"0" maximum:"2200"`
		MinRating  float64  `query:"minRating" minimum:"0" maximum:"10"`
		MaxRuntime int      `query:"maxRuntime" minimum:"0" maximum:"1000" doc:"Minutes"`
		Watched    string   `query:"watched" enum:"any,yes,no" default:"any" doc:"By the logged-in user"`
		Nobody     bool     `query:"neverWatched" doc:"Admins only: titles nobody ever watched (library clean-up)"`
		Sort       string   `query:"sort" enum:"added,title,year,rating,runtime,popular" default:"added"`
		Dir        string   `query:"dir" enum:"asc,desc" default:"desc"`
		Page       int      `query:"page" minimum:"1" default:"1"`
		PageSize   int      `query:"pageSize" minimum:"1" maximum:"100" default:"48"`
	}) (*struct{ Body libraryList }, error) {
		u, err := requireUser(ctx)
		if err != nil {
			return nil, err
		}
		if in.Nobody && u.Role != store.RoleAdmin {
			return nil, fail(403, "admin_only", "admin only")
		}
		f := storeFilter(u.ID, in.Type, in.Q, in.Genre, in.YearFrom, in.YearTo, int(in.MinRating*10+0.5), in.MaxRuntime, in.Watched, in.Sort, in.Dir == "desc", (in.Page-1)*in.PageSize, in.PageSize)
		f.Nobody = in.Nobody
		rows, total, err := d.Store.ListLibrary(ctx, f)
		if err != nil {
			return nil, err
		}
		out := libraryList{Total: total, Page: in.Page, TotalPages: (total + in.PageSize - 1) / in.PageSize, Items: make([]libraryItem, 0, len(rows))}
		for _, r := range rows {
			it := libraryItem{
				Type: r.MediaType, TMDBID: r.TMDBID, Title: r.Title, Year: r.Year, RuntimeMin: r.RuntimeMin, Rating: float64(r.Rating10) / 10,
				Genres: []string{}, AddedAt: r.AddedAt, Plays: r.Plays, Viewers: r.Viewers, Watched: r.MyPlays > 0,
			}
			if g := strings.Trim(r.Genres, "|"); g != "" {
				it.Genres = strings.Split(g, "|")
			}
			if r.ImageTag != "" {
				it.PosterURL = "/api/v1/images/jellyfin/" + r.JellyfinID + "?tag=" + r.ImageTag
			}
			out.Items = append(out.Items, it)
		}
		return &struct{ Body libraryList }{out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "libraryFacets", Method: http.MethodGet, Path: "/library/facets",
		Summary: "Genres (with counts), years and counts to build the Library filters", Tags: []string{"library"}, Security: sec, Errors: []int{http.StatusUnauthorized},
	}, func(ctx context.Context, _ *struct{}) (*struct{ Body libraryFacets }, error) {
		if _, err := requireUser(ctx); err != nil {
			return nil, err
		}
		f, err := d.Store.LibraryFacets(ctx)
		if err != nil {
			return nil, err
		}
		out := libraryFacets{Movies: f.Movies, Shows: f.Shows, YearMin: f.YearMin, YearMax: f.YearMax, MaxRuntime: f.MaxRuntime, Genres: make([]genreCount, 0, len(f.Genres))}
		for _, g := range f.Genres {
			out.Genres = append(out.Genres, genreCount{g.Name, g.Count})
		}
		return &struct{ Body libraryFacets }{out}, nil
	})
}

// jellyfinImageHandler serves a title's Jellyfin poster through a disk cache, so the browser never
// needs Jellyfin's address or an API key. The tag (Jellyfin's image tag) makes the URL immutable.
func jellyfinImageHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if userFrom(r.Context()) == nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		id, tag := chiParam(r, "id"), r.URL.Query().Get("tag")
		if !jfIDRe.MatchString(id) || !jfTagRe.MatchString(tag) {
			http.NotFound(w, r)
			return
		}
		cached := filepath.Join(d.ConfigDir, "cache", "jellyfin", strings.ToLower(id)+"-"+tag)
		if _, err := os.Stat(cached); err != nil {
			body, ct, err := d.Library.Image(r.Context(), id, tag)
			if err != nil || !strings.HasPrefix(ct, "image/") {
				http.NotFound(w, r)
				return
			}
			if err := os.MkdirAll(filepath.Dir(cached), 0o755); err != nil {
				http.Error(w, "image cache write failed", http.StatusInternalServerError)
				return
			}
			tmp, err := os.CreateTemp(filepath.Dir(cached), ".dl-*")
			if err != nil {
				http.Error(w, "image cache write failed", http.StatusInternalServerError)
				return
			}
			_, werr := tmp.Write(body)
			cerr := tmp.Close()
			if werr != nil || cerr != nil || os.Rename(tmp.Name(), cached) != nil {
				os.Remove(tmp.Name())
				http.Error(w, "image cache write failed", http.StatusInternalServerError)
				return
			}
		}
		if tag != "" {
			w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "private, max-age=3600")
		}
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeFile(w, r, cached)
	}
}

func storeFilter(userID, mediaType, q string, genres []string, yearFrom, yearTo, minRating, maxRuntime int, watched, sort string, desc bool, offset, limit int) store.LibraryFilter {
	if mediaType == "all" {
		mediaType = ""
	}
	if watched == "any" {
		watched = ""
	}
	return store.LibraryFilter{
		UserID: userID, MediaType: mediaType, Query: q, Genres: genres, YearFrom: yearFrom, YearTo: yearTo,
		MinRating: minRating, MaxRuntime: maxRuntime, Watched: watched, Sort: sort, Desc: desc, Offset: offset, Limit: limit,
	}
}
