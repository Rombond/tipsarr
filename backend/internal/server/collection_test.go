package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/perms"
)

type collectionResult struct {
	Requested []reqView `json:"requested"`
	Skipped   []struct {
		TMDBID int    `json:"tmdbId"`
		Title  string `json:"title"`
		Code   string `json:"code"`
	} `json:"skipped"`
}

func requestCollection(t *testing.T, e *env, who *http.Cookie, body string) collectionResult {
	t.Helper()
	resp, body := call(t, e.app, "POST", "/api/v1/collection/9/request", body, who)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("collection request = %d %s", resp.StatusCode, body)
	}
	var r collectionResult
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return r
}

// "Request missing" on a collection makes one normal request per released part that is neither in the
// library nor requested; picked parts may include an announced film; foreign ids are refused.
func TestRequestCollection(t *testing.T) {
	t.Parallel()
	e := newEnv(t, true)
	_, _, bob := setupUsers(t, e)

	r := requestCollection(t, e, bob, `{}`)
	if len(r.Requested) != 2 || len(r.Skipped) != 0 {
		t.Fatalf("first run = %+v", r)
	}
	for _, v := range r.Requested {
		if v.Status != "pending" { // bob does not auto-approve: the normal flow, no bypass
			t.Fatalf("request %q status = %s", v.Title, v.Status)
		}
	}
	if r.Requested[0].Title != "Movie One" || r.Requested[1].Title != "Request Me" {
		t.Fatalf("order = %q, %q", r.Requested[0].Title, r.Requested[1].Title)
	}

	// the parts now show as requested; asking again requests nothing
	resp, body := call(t, e.app, "GET", "/api/v1/collection/9", "", bob)
	var c struct {
		Parts []struct {
			TMDBID        int    `json:"tmdbId"`
			RequestStatus string `json:"requestStatus"`
		} `json:"parts"`
	}
	if resp.StatusCode != 200 || json.Unmarshal([]byte(body), &c) != nil {
		t.Fatalf("collection = %d %s", resp.StatusCode, body)
	}
	for _, p := range c.Parts {
		if (p.TMDBID == 1 || p.TMDBID == 5) != (p.RequestStatus == "pending") {
			t.Fatalf("part %d status %q", p.TMDBID, p.RequestStatus)
		}
	}
	if r := requestCollection(t, e, bob, `{}`); len(r.Requested) != 0 || len(r.Skipped) != 0 {
		t.Fatalf("second run = %+v", r)
	}

	// picked parts: the announced one is allowed, an already requested one and a stranger are skipped
	r = requestCollection(t, e, bob, `{"tmdbIds":[7,5,404]}`)
	if len(r.Requested) != 1 || r.Requested[0].Title != "Coming Soon" || len(r.Skipped) != 2 ||
		r.Skipped[0].Code != "duplicate_request" || r.Skipped[1].Code != "not_in_collection" {
		t.Fatalf("picked = %+v", r)
	}
}

// The quota counts every part: once it is used up the remaining parts are skipped with quota_reached.
func TestRequestCollectionQuota(t *testing.T) {
	t.Parallel()
	e := newEnv(t, true)
	_, admin, bob := setupUsers(t, e)
	call(t, e.app, "PUT", "/api/v1/admin/permissions/defaults", fmt.Sprintf(`{"permissions":%d,"movieQuotaLimit":1,"movieQuotaDays":7,"tvQuotaLimit":0,"tvQuotaDays":0}`, perms.Default), admin)

	r := requestCollection(t, e, bob, `{}`)
	if len(r.Requested) != 1 || len(r.Skipped) != 1 || r.Skipped[0].Code != "quota_reached" || r.Skipped[0].Title != "Request Me" {
		t.Fatalf("with a quota of one = %+v", r)
	}

	// no right to request movies: refused as a whole
	call(t, e.app, "PATCH", "/api/v1/admin/users/"+userID(t, e, bob), fmt.Sprintf(`{"permissions":%d}`, perms.RequestTV), admin)
	if resp, body := call(t, e.app, "POST", "/api/v1/collection/9/request", `{}`, bob); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("without the movie right = %d %s", resp.StatusCode, body)
	}
}
