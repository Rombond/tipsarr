// Package notify delivers outgoing webhooks (ntfy, Discord, anything that takes a JSON POST).
package notify

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/netguard"
	"github.com/Rombond/tipsarr/backend/internal/store"
)

// Event names sent to webhooks.
const (
	RequestCreated  = "request.created"
	RequestApproved = "request.approved"
	RequestDeclined = "request.declined"
	RequestFailed   = "request.failed"
	MediaAvailable  = "media.available"
	IssueCreated    = "issue.created"
	IssueCommented  = "issue.commented"
	IssueResolved   = "issue.resolved"
	// RequestAutoApproved replaces request.approved when the requester's rights approved it at once.
	// A webhook subscribed to request.approved receives it too (compatibility).
	RequestAutoApproved = "request.auto_approved"
	IssueReopened       = "issue.reopened"
	// Test is only sent by the test button (it ignores the subscriptions).
	Test = "test"
)

// AllEvents are the events a webhook may subscribe to.
var AllEvents = []string{RequestCreated, RequestApproved, RequestAutoApproved, RequestDeclined, RequestFailed, MediaAvailable, IssueCreated, IssueCommented, IssueResolved, IssueReopened}

// PushEvent is what a mobile push is about. The relay turns the event name into a localized alert
// and the app fetches the details with ID, so nothing else travels.
type PushEvent struct {
	Name   string   // relay event, e.g. request.approved
	ID     string   // request or issue id, the app opens it
	Users  []string // user ids to notify
	Admins bool     // also every admin, and whoever has one of the Needs flags
	Needs  int64    // permission bits (package perms) that make someone an "admin" for this alert, 0 = the role only
	Skip   string   // user id that caused the event (never notified about their own action)
	// Subject is the title of the media the event is about. It travels to the relay only when the admin
	// turned on "alert text" (push.text); otherwise the relay's generic alert is shown.
	Subject string
}

// Pusher delivers push events (implemented by package push).
type Pusher interface{ Send(PushEvent) }

type Service struct {
	pusher Pusher
	store  *store.Store
	http   *http.Client
	dryRun bool
	// Offline (demo mode) refuses every delivery: nothing ever leaves the process.
	Offline bool
	delays  []time.Duration // waits before retry 2, 3, ...
}

// SettingAllowedHosts lists the private-network targets webhooks may reach (one per line or comma:
// host, host:port or CIDR). Loopback and link-local addresses are refused whatever it says.
const SettingAllowedHosts = "webhooks.allowed_internal_hosts"

func New(s *store.Store, dryRun bool) *Service {
	svc := &Service{store: s, dryRun: dryRun, delays: []time.Duration{time.Second, 5 * time.Second}}
	svc.http = svc.Policy().Client(10*time.Second, 3)
	return svc
}

// Policy is the egress policy of webhook deliveries: public addresses, plus the private ones the
// admin allowed (SettingAllowedHosts).
func (s *Service) Policy() netguard.Policy {
	return netguard.Policy{AllowList: func() []string {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, _ := s.store.GetSetting(ctx, SettingAllowedHosts)
		return SplitAllowList(v)
	}}
}

// SplitAllowList splits the stored allow-list (lines or commas).
func SplitAllowList(v string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == '\n' || r == ',' || r == '\r' }) {
		if f = strings.TrimSpace(f); f != "" {
			out = append(out, f)
		}
	}
	return out
}

// SetPusher enables mobile push; without it Push does nothing.
func (s *Service) SetPusher(p Pusher) { s.pusher = p }

// Push hands a mobile push event to the pusher, if any. It never blocks.
func (s *Service) Push(ev PushEvent) {
	if s.pusher != nil {
		s.pusher.Send(ev)
	}
}

// SetRetryDelays overrides the retry schedule (tests).
func (s *Service) SetRetryDelays(d ...time.Duration) { s.delays = d }
