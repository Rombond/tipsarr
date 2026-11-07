package store

import "testing"

func TestApplyGrant(t *testing.T) {
	pw := AdminGrant{Source: AdminFromJellyfin, Revokes: []string{AdminFromJellyfin}}
	sso := AdminGrant{Source: AdminFromOIDC, Revokes: []string{AdminFromJellyfin, AdminFromOIDC}}
	yes := func(g AdminGrant) AdminGrant { g.Admin = true; return g }
	cases := []struct {
		name             string
		role, source     string
		g                AdminGrant
		wantRole, wantSr string
	}{
		{"promote", RoleUser, "", yes(pw), RoleAdmin, AdminFromJellyfin},
		{"jellyfin revoked", RoleAdmin, AdminFromJellyfin, pw, RoleUser, ""},
		{"oidc kept by a password sign-in", RoleAdmin, AdminFromOIDC, pw, RoleAdmin, AdminFromOIDC},
		{"oidc revoked by sso", RoleAdmin, AdminFromOIDC, sso, RoleUser, ""},
		{"manual kept", RoleAdmin, AdminFromManual, sso, RoleAdmin, AdminFromManual},
		{"manual stays manual when granted", RoleAdmin, AdminFromManual, yes(pw), RoleAdmin, AdminFromManual},
		{"legacy granted: tracked", RoleAdmin, "", yes(pw), RoleAdmin, AdminFromJellyfin},
		{"legacy not granted: kept as manual", RoleAdmin, "", pw, RoleAdmin, AdminFromManual},
		{"plain user", RoleUser, "", pw, RoleUser, ""},
	}
	for _, c := range cases {
		u := &User{Name: "x", Role: c.role, AdminSource: c.source}
		applyGrant(u, c.g)
		if u.Role != c.wantRole || u.AdminSource != c.wantSr {
			t.Errorf("%s: got %s/%s, want %s/%s", c.name, u.Role, u.AdminSource, c.wantRole, c.wantSr)
		}
	}
}
