package jellyfin

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// Item is the subset of Jellyfin's BaseItemDto Tipsarr needs.
type Item struct {
	ID                string            `json:"Id"`
	Name              string            `json:"Name"`
	Type              string            `json:"Type"` // Movie | Series | Episode
	ProviderIDs       map[string]string `json:"ProviderIds"`
	SeriesID          string            `json:"SeriesId"`
	ParentIndexNumber int               `json:"ParentIndexNumber"` // season number (episodes)
	IndexNumber       int               `json:"IndexNumber"`       // episode number
	UserData          struct {
		Played         bool   `json:"Played"`
		PlayCount      int    `json:"PlayCount"`
		LastPlayedDate string `json:"LastPlayedDate"` // RFC 3339
	} `json:"UserData"`
}

// TMDBID returns the item's TMDB id, or 0 when Jellyfin has none.
func (i Item) TMDBID() int {
	for k, v := range i.ProviderIDs {
		if k == "Tmdb" || k == "tmdb" {
			n, _ := strconv.Atoi(v)
			return n
		}
	}
	return 0
}

type User struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
}

// Users lists all Jellyfin users (needs an API key).
func (c *Client) Users(ctx context.Context) ([]User, error) {
	var out []User
	err := c.do(ctx, http.MethodGet, "/Users", nil, &out)
	return out, err
}

const pageSize = 500

// Items pages through /Items (or /Users/{id}/Items when userID is set), calling fn for each item.
func (c *Client) Items(ctx context.Context, userID string, q url.Values, fn func(Item) error) error {
	path := "/Items"
	if userID != "" {
		path = "/Users/" + url.PathEscape(userID) + "/Items"
	}
	q = cloneValues(q)
	q.Set("Recursive", "true")
	q.Set("Limit", strconv.Itoa(pageSize))
	for start := 0; ; start += pageSize {
		q.Set("StartIndex", strconv.Itoa(start))
		var page struct {
			Items            []Item `json:"Items"`
			TotalRecordCount int    `json:"TotalRecordCount"`
		}
		if err := c.do(ctx, http.MethodGet, path+"?"+q.Encode(), nil, &page); err != nil {
			return err
		}
		for _, it := range page.Items {
			if err := fn(it); err != nil {
				return err
			}
		}
		if len(page.Items) < pageSize {
			return nil
		}
	}
}

func cloneValues(q url.Values) url.Values {
	out := url.Values{}
	for k, v := range q {
		out[k] = append([]string(nil), v...)
	}
	return out
}
