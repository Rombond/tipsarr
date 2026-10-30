package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

// settingAPIKey holds the read-only API key for dashboards (Glance, Homepage...). Empty = off.
const settingAPIKey = "api.key"

// apiKeyHeader is where the key travels (like Radarr and Sonarr). Never the URL: addresses end up in proxy logs.
const apiKeyHeader = "X-Api-Key"

// apiKeyUserID is the id of the stand-in account the key acts as.
const apiKeyUserID = "api-key"

func newAPIKey() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

// updateAPIKey turns the dashboard key on or off and replaces it on request. Off deletes the key; on creates one
// when there is none. Regenerating while off turns it on.
func updateAPIKey(ctx context.Context, d Deps, enabled, regenerate *bool) error {
	saved, _ := d.Store.GetSetting(ctx, settingAPIKey)
	switch {
	case enabled != nil && !*enabled:
		if saved == "" {
			return nil
		}
		return d.Store.SetSetting(ctx, settingAPIKey, "")
	case (regenerate != nil && *regenerate) || (enabled != nil && *enabled && saved == ""):
		key, err := newAPIKey()
		if err != nil {
			return err
		}
		return d.Store.SetSetting(ctx, settingAPIKey, key)
	}
	return nil
}

// apiKeyUser is the account a valid key acts as: an admin who can read everything. The middleware refuses every
// method that could change something, so the key never writes.
func apiKeyUser() *store.User {
	return (&store.User{ID: apiKeyUserID, Name: "API key", Role: store.RoleAdmin}).Resolve()
}

// apiKeyMatches compares the request header with the saved key in constant time.
func apiKeyMatches(r *http.Request, saved string) bool {
	given := r.Header.Get(apiKeyHeader)
	return saved != "" && given != "" && subtle.ConstantTimeCompare([]byte(given), []byte(saved)) == 1
}

func readOnlyMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}
