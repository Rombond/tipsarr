package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/perms"
)

// userID reads the id of a signed-in person.
func userID(t *testing.T, e *env, who *http.Cookie) string {
	t.Helper()
	_, body := call(t, e.app, "GET", "/api/v1/me", "", who)
	var me struct{ ID string }
	if err := json.Unmarshal([]byte(body), &me); err != nil || me.ID == "" {
		t.Fatalf("me = %s", body)
	}
	return me.ID
}

// grantPermissions sets someone's mask as an admin would.
func grantPermissions(t *testing.T, e *env, admin, who *http.Cookie, mask perms.Mask) {
	t.Helper()
	resp, body := call(t, e.app, "PATCH", "/api/v1/admin/users/"+userID(t, e, who), fmt.Sprintf(`{"permissions":%d}`, mask), admin)
	if resp.StatusCode != 200 {
		t.Fatalf("grant %d = %d %s", mask, resp.StatusCode, body)
	}
}

func effective(t *testing.T, e *env, who *http.Cookie) perms.Mask {
	t.Helper()
	_, body := call(t, e.app, "GET", "/api/v1/me", "", who)
	var me struct {
		Permissions          int64 `json:"permissions"`
		EffectivePermissions int64 `json:"effectivePermissions"`
	}
	if err := json.Unmarshal([]byte(body), &me); err != nil {
		t.Fatalf("me = %s", body)
	}
	return perms.Mask(me.EffectivePermissions)
}

func TestDefaultPermissionsAndTheCatalog(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	if got := effective(t, e, bob); got != perms.Default {
		t.Fatalf("a new person has %d, want the default %d", got, perms.Default)
	}
	if got := effective(t, e, admin); got&perms.ManageUsers == 0 || got&perms.Admin == 0 {
		t.Fatalf("an admin must have every flag, has %d", got)
	}
	// the catalog is for people who manage users
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/permissions", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("catalog for a plain user = %d", resp.StatusCode)
	}
	resp, body := call(t, e.app, "GET", "/api/v1/admin/permissions", "", admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"manageRequests"`) || !strings.Contains(body, `"reserved":true`) || strings.Contains(body, `"name":"admin"`) {
		t.Fatalf("catalog = %d %s", resp.StatusCode, body)
	}
	// defaults for new people are saved by admins only
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageUsers)
	if resp, _ := call(t, e.app, "PUT", "/api/v1/admin/permissions/defaults", fmt.Sprintf(`{"permissions":%d,"movieQuotaLimit":0,"movieQuotaDays":0,"tvQuotaLimit":0,"tvQuotaDays":0}`, perms.Default), bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("defaults by a user manager = %d", resp.StatusCode)
	}
	resp, body = call(t, e.app, "PUT", "/api/v1/admin/permissions/defaults", fmt.Sprintf(`{"permissions":%d,"movieQuotaLimit":3,"movieQuotaDays":7,"tvQuotaLimit":0,"tvQuotaDays":0}`, perms.Default|perms.AutoApproveMovie), admin)
	if resp.StatusCode != 200 || !strings.Contains(body, `"movieQuotaLimit":3`) {
		t.Fatalf("defaults = %d %s", resp.StatusCode, body)
	}
	if got := e.st.DefaultPermissions(t.Context()); got&perms.AutoApproveMovie == 0 {
		t.Fatalf("stored default = %d", got)
	}
}

func TestPermissionEditingRules(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	bobID := userID(t, e, bob)
	adminID := userID(t, e, admin)

	// the admin bit is never stored: nobody becomes admin through a mask
	grantPermissions(t, e, admin, bob, perms.Request|perms.Admin)
	if got := effective(t, e, bob); got&perms.Admin != 0 || got&perms.ManageUsers != 0 {
		t.Fatalf("bob got admin through a mask: %d", got)
	}
	// a person with only the user-management right can hand out day-to-day rights, not the keys
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageUsers)
	resp, body := call(t, e.app, "GET", "/api/v1/admin/users", "", bob)
	if resp.StatusCode != 200 {
		t.Fatalf("user manager lists users = %d %s", resp.StatusCode, body)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+adminID, `{"role":"user"}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a user manager changed a role = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, fmt.Sprintf(`{"permissions":%d}`, perms.Default), bob); resp.StatusCode != http.StatusConflict {
		t.Fatalf("own permissions = %d", resp.StatusCode)
	}
	// the other person bob can edit is the admin account: day-to-day rights are fine, the keys are not
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+adminID, fmt.Sprintf(`{"permissions":%d}`, perms.Default|perms.ManageSettings), bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("a user manager granted the settings key = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "PATCH", "/api/v1/admin/users/"+adminID, fmt.Sprintf(`{"permissions":%d}`, perms.Default|perms.AutoApprove), bob); resp.StatusCode != 200 {
		t.Fatalf("a user manager granted auto-approve = %d", resp.StatusCode)
	}
	// admin-only endpoints follow the settings right
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/settings", "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("settings without the right = %d", resp.StatusCode)
	}
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageSettings)
	if resp, _ := call(t, e.app, "GET", "/api/v1/admin/settings", "", bob); resp.StatusCode != 200 {
		t.Fatalf("settings with the right = %d", resp.StatusCode)
	}
}

func TestApprovalFollowsPermissions(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	addInstance(t, e, admin, "radarr", newFakeArr(t, "radarr"))
	addInstance(t, e, admin, "sonarr", newFakeArr(t, "sonarr"))

	// by default a request waits, and a person without the right cannot approve
	_, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob)
	v := decodeReq(t, body)
	if v.Status != "pending" {
		t.Fatalf("default request = %s", body)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("self-approval = %d", resp.StatusCode)
	}
	// someone who manages requests approves (and sees everyone's requests)
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageRequests)
	if resp, body := call(t, e.app, "POST", "/api/v1/requests/"+v.ID+"/approve", `{}`, bob); resp.StatusCode != 200 || decodeReq(t, body).Status != "approved" {
		t.Fatalf("manager approval = %d %s", resp.StatusCode, body)
	}
	// auto-approve per type
	grantPermissions(t, e, admin, bob, perms.Default|perms.AutoApproveMovie)
	_, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":1}`, bob)
	if v := decodeReq(t, body); v.Status != "approved" {
		t.Fatalf("auto-approved movie = %s", body)
	}
	_, body = call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2}`, bob)
	if v := decodeReq(t, body); v.Status != "pending" {
		t.Fatalf("shows are not auto-approved by the movie right = %s", body)
	}
}

func TestRequestRights(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	grantPermissions(t, e, admin, bob, perms.CreateIssues) // may not request at all
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("request without the right = %d", resp.StatusCode)
	}
	grantPermissions(t, e, admin, bob, perms.RequestMovie)
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"tv","tmdbId":2}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("show request with the movie right only = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusCreated {
		t.Fatalf("movie request with the movie right = %d", resp.StatusCode)
	}
	// seeing everyone's requests is a right too
	_, body := call(t, e.app, "GET", "/api/v1/requests", "", admin)
	if !strings.Contains(body, `"total":1`) {
		t.Fatalf("admin list = %s", body)
	}
	other := userID(t, e, admin)
	_ = other
	if _, body := call(t, e.app, "GET", "/api/v1/requests", "", bob); !strings.Contains(body, `"total":1`) {
		t.Fatalf("own list = %s", body)
	}
}

func TestQuota(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	e.reqs.SetClock(func() time.Time { return now })
	bobID := userID(t, e, bob)

	quota := func(who *http.Cookie) (movie struct {
		Limit, Used int
		Remaining   *int
		Restricted  bool
		ResetsAt    int64
	}) {
		t.Helper()
		_, body := call(t, e.app, "GET", "/api/v1/me/quota", "", who)
		var q struct {
			Movie struct {
				Limit, Used int
				Remaining   *int
				Restricted  bool
				ResetsAt    int64
			} `json:"movie"`
		}
		if err := json.Unmarshal([]byte(body), &q); err != nil {
			t.Fatalf("quota = %s", body)
		}
		return q.Movie
	}
	if q := quota(bob); q.Limit != 0 || q.Restricted {
		t.Fatalf("no limit by default: %+v", q)
	}

	// a limit of one movie a week, from the defaults
	call(t, e.app, "PUT", "/api/v1/admin/permissions/defaults", fmt.Sprintf(`{"permissions":%d,"movieQuotaLimit":1,"movieQuotaDays":7,"tvQuotaLimit":0,"tvQuotaDays":0}`, perms.Default), admin)
	if q := quota(bob); q.Limit != 1 || q.Used != 0 || q.Remaining == nil || *q.Remaining != 1 {
		t.Fatalf("quota = %+v", q)
	}
	if resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusCreated {
		t.Fatalf("first request = %d %s", resp.StatusCode, body)
	}
	q := quota(bob)
	if !q.Restricted || q.Used != 1 || q.ResetsAt != now.Unix()+7*86400 && q.ResetsAt == 0 {
		t.Fatalf("quota after one request = %+v", q)
	}
	resp, body := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":1}`, bob)
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(body, "quota_reached") {
		t.Fatalf("second request = %d %s", resp.StatusCode, body)
	}
	// admins and managers have no limit; a person's own quota replaces the default
	if q := quota(admin); q.Limit != 0 {
		t.Fatalf("admin quota = %+v", q)
	}
	call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, `{"quota":{"movieLimit":5,"movieDays":7}}`, admin)
	if q := quota(bob); q.Limit != 5 || q.Restricted {
		t.Fatalf("own quota = %+v", q)
	}
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":1}`, bob); resp.StatusCode != http.StatusCreated {
		t.Fatalf("request under the own quota = %d", resp.StatusCode)
	}
	// clearing it goes back to the default (one a week, already used up twice)
	call(t, e.app, "PATCH", "/api/v1/admin/users/"+bobID, `{"quota":{}}`, admin)
	if q := quota(bob); q.Limit != 1 || !q.Restricted || q.Used != 2 {
		t.Fatalf("cleared quota = %+v", q)
	}
	// a declined request gives its share back
	_, list := call(t, e.app, "GET", "/api/v1/requests?filter=pending", "", admin)
	var l struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	_ = json.Unmarshal([]byte(list), &l)
	for _, it := range l.Items {
		call(t, e.app, "POST", "/api/v1/requests/"+it.ID+"/decline", `{"reason":"no"}`, admin)
	}
	if q := quota(bob); q.Used != 0 || q.Restricted {
		t.Fatalf("declined requests still count: %+v", q)
	}
	// the window moves: requests older than the period no longer count
	if resp, _ := call(t, e.app, "POST", "/api/v1/requests", `{"type":"movie","tmdbId":5}`, bob); resp.StatusCode != http.StatusCreated {
		t.Fatalf("request after the decline = %d", resp.StatusCode)
	}
	if q := quota(bob); !q.Restricted {
		t.Fatalf("quota = %+v", q)
	}
	now = now.Add(8 * 24 * time.Hour)
	if q := quota(bob); q.Used != 0 || q.Restricted {
		t.Fatalf("quota after the window = %+v", q)
	}
}

func TestIssueRights(t *testing.T) {
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	grantPermissions(t, e, admin, bob, perms.Request) // no issue rights
	if resp, _ := call(t, e.app, "POST", "/api/v1/issues", `{"type":"movie","tmdbId":5,"kind":"video","message":"broken"}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("issue without the right = %d", resp.StatusCode)
	}
	grantPermissions(t, e, admin, bob, perms.Default)
	resp, body := call(t, e.app, "POST", "/api/v1/issues", `{"type":"movie","tmdbId":5,"kind":"video","message":"broken"}`, bob)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("issue = %d %s", resp.StatusCode, body)
	}
	var iss struct{ ID string }
	_ = json.Unmarshal([]byte(body), &iss)
	// the reporter may resolve their own; a person with only the view right may see but not act on someone else's
	if resp, _ := call(t, e.app, "POST", "/api/v1/issues/"+iss.ID+"/resolve", `{}`, bob); resp.StatusCode != 200 {
		t.Fatalf("resolve own = %d", resp.StatusCode)
	}
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/issues/"+iss.ID, "", bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("delete without manage = %d", resp.StatusCode)
	}
	grantPermissions(t, e, admin, bob, perms.Default|perms.ManageIssues)
	if resp, _ := call(t, e.app, "DELETE", "/api/v1/issues/"+iss.ID, "", bob); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete with manage = %d", resp.StatusCode)
	}
}
