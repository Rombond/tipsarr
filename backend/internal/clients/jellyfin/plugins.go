package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Plugin is an installed Jellyfin plugin.
type Plugin struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Status string `json:"Status"`
}

// Plugins lists the installed plugins (needs an API key).
func (c *Client) Plugins(ctx context.Context) ([]Plugin, error) {
	var out []Plugin
	err := c.do(ctx, http.MethodGet, "/Plugins", nil, &out)
	return out, err
}

// LDAPConfig is the part of the "LDAP Authentication" plugin's settings Tipsarr can reuse.
type LDAPConfig struct {
	Server       string `json:"LdapServer"`
	Port         int    `json:"LdapPort"`
	UseSSL       bool   `json:"UseSsl"`
	BindUser     string `json:"LdapBindUser"`
	BindPassword string `json:"LdapBindPassword"`
	BaseDN       string `json:"LdapBaseDn"`
}

// URL is the address in the form Tipsarr stores it: ldap://host:port or ldaps://host:port.
func (l LDAPConfig) URL() string {
	scheme, port := "ldap", l.Port
	if l.UseSSL {
		scheme = "ldaps"
	}
	if port == 0 {
		port = 389
		if l.UseSSL {
			port = 636
		}
	}
	return scheme + "://" + l.Server + ":" + strconv.Itoa(port)
}

// LDAPPlugin returns the settings of the LDAP Authentication plugin, or nil when it is not installed.
func (c *Client) LDAPPlugin(ctx context.Context) (*LDAPConfig, error) {
	plugins, err := c.Plugins(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range plugins {
		if !strings.Contains(strings.ToLower(p.Name), "ldap") {
			continue
		}
		var cfg LDAPConfig
		if err := c.do(ctx, http.MethodGet, "/Plugins/"+url.PathEscape(p.ID)+"/Configuration", nil, &cfg); err != nil {
			return nil, err
		}
		if cfg.Server == "" {
			return nil, nil
		}
		return &cfg, nil
	}
	return nil, nil
}

// PlaybackReportingInstalled reports whether the Playback Reporting plugin is installed.
func (c *Client) PlaybackReportingInstalled(ctx context.Context) (bool, error) {
	plugins, err := c.Plugins(ctx)
	if err != nil {
		return false, err
	}
	for _, p := range plugins {
		if strings.Contains(strings.ToLower(p.Name), "playback reporting") {
			return true, nil
		}
	}
	return false, nil
}

// PlaybackQuery runs a SELECT on the Playback Reporting plugin's database and returns the column
// names and the rows (every value as text). WARNING: the plugin runs ANY SQL it is sent, with
// write access, so the query must always be built by Tipsarr itself and never contain text
// that came from a person.
func (c *Client) PlaybackQuery(ctx context.Context, query string) ([]string, [][]string, error) {
	body, err := json.Marshal(map[string]any{"CustomQueryString": query, "ReplaceUserId": false})
	if err != nil {
		return nil, nil, err
	}
	var out struct {
		Columns []string   `json:"colums"` // sic: the plugin's spelling
		Results [][]string `json:"results"`
		Message string     `json:"message"`
	}
	if err := c.do(ctx, http.MethodPost, "/user_usage_stats/submit_custom_query", body, &out); err != nil {
		return nil, nil, err
	}
	if strings.Contains(out.Message, "Error Running Query") {
		return nil, nil, errors.New("Playback Reporting could not run the query")
	}
	return out.Columns, out.Results, nil
}

// ItemsByID fetches items by Jellyfin id (episodes: the show they belong to is in SeriesID).
func (c *Client) ItemsByID(ctx context.Context, ids []string, fn func(Item) error) error {
	return c.Items(ctx, "", url.Values{"Ids": {strings.Join(ids, ",")}}, fn)
}
