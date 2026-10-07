package jellyfin

import (
	"context"
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
