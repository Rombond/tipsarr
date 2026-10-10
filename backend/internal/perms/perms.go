// Package perms is Tipsarr's permission model, taken from Seerr: one integer mask per person, a flag per right,
// and an admin who may do everything. The bit values are Seerr's, so a mask can be copied across as it is
// (4K and voting have no counterpart here and are left out).
package perms

import "slices"

// Mask is a set of flags stored as bits.
type Mask int64

const (
	None Mask = 0

	// Admin and management.
	Admin            Mask = 2
	ManageSettings   Mask = 4   // server settings, Radarr/Sonarr, webhooks, jobs
	ManageUsers      Mask = 8   // the user list, roles, permissions
	ManageRequests   Mask = 16  // approve, decline, retry, edit and delete any request; no quota; implies auto-approve
	Request          Mask = 32  // make requests at all
	AutoApprove      Mask = 128 // requests are approved without an admin
	AutoApproveMovie Mask = 256
	AutoApproveTV    Mask = 512
	RequestAdvanced  Mask = 8192    // choose the root folder of a request
	RequestView      Mask = 16384   // see everyone's requests
	RequestMovie     Mask = 262144  // request movies
	RequestTV        Mask = 524288  // request shows
	ManageIssues     Mask = 1048576 // comment on and resolve any issue
	ViewIssues       Mask = 2097152 // see everyone's issues
	CreateIssues     Mask = 4194304 // report a problem with a title

	// Reserved: part of the mask so the numbers match Seerr, but no feature uses them yet.
	AutoRequest      Mask = 8388608
	AutoRequestMovie Mask = 16777216
	AutoRequestTV    Mask = 33554432
	RecentView       Mask = 67108864
	WatchlistView    Mask = 134217728
	ManageBlocklist  Mask = 268435456
	ViewBlocklist    Mask = 1073741824
)

// Flag describes one right for the admin screens.
type Flag struct {
	Name     string   `json:"name"`
	Value    Mask     `json:"value"`
	Group    string   `json:"group" enum:"admin,requests,auto-approve,issues,reserved"`
	Implies  []string `json:"implies,omitempty" doc:"Flags that come with this one and cannot be switched off while it is on"`
	Reserved bool     `json:"reserved" doc:"Defined so the numbers match Seerr; no feature uses it yet, hide it"`
}

// Catalog lists every flag in the order the admin screens show them.
var Catalog = []Flag{
	{Name: "manageSettings", Value: ManageSettings, Group: "admin"},
	{Name: "manageUsers", Value: ManageUsers, Group: "admin"},
	{Name: "manageRequests", Value: ManageRequests, Group: "requests", Implies: []string{"requestAdvanced", "requestView", "autoApprove", "autoApproveMovie", "autoApproveTV"}},
	{Name: "request", Value: Request, Group: "requests", Implies: []string{"requestMovie", "requestTV"}},
	{Name: "requestMovie", Value: RequestMovie, Group: "requests"},
	{Name: "requestTV", Value: RequestTV, Group: "requests"},
	{Name: "requestAdvanced", Value: RequestAdvanced, Group: "requests"},
	{Name: "requestView", Value: RequestView, Group: "requests"},
	{Name: "autoApprove", Value: AutoApprove, Group: "auto-approve", Implies: []string{"autoApproveMovie", "autoApproveTV"}},
	{Name: "autoApproveMovie", Value: AutoApproveMovie, Group: "auto-approve"},
	{Name: "autoApproveTV", Value: AutoApproveTV, Group: "auto-approve"},
	{Name: "manageIssues", Value: ManageIssues, Group: "issues", Implies: []string{"viewIssues", "createIssues"}},
	{Name: "viewIssues", Value: ViewIssues, Group: "issues"},
	{Name: "createIssues", Value: CreateIssues, Group: "issues"},
	{Name: "autoRequest", Value: AutoRequest, Group: "reserved", Reserved: true, Implies: []string{"autoRequestMovie", "autoRequestTV"}},
	{Name: "autoRequestMovie", Value: AutoRequestMovie, Group: "reserved", Reserved: true},
	{Name: "autoRequestTV", Value: AutoRequestTV, Group: "reserved", Reserved: true},
	{Name: "recentView", Value: RecentView, Group: "reserved", Reserved: true},
	{Name: "watchlistView", Value: WatchlistView, Group: "reserved", Reserved: true},
	{Name: "manageBlocklist", Value: ManageBlocklist, Group: "reserved", Reserved: true, Implies: []string{"viewBlocklist"}},
	{Name: "viewBlocklist", Value: ViewBlocklist, Group: "reserved", Reserved: true},
}

// All is every flag, the admin bit included.
var All = func() Mask {
	m := Admin
	for _, f := range Catalog {
		m |= f.Value
	}
	return m
}()

// Default is what a new person gets: request movies and shows and report problems, nothing auto-approved.
const Default = Request | RequestMovie | RequestTV | CreateIssues

// Has reports whether every given flag is set. Admin passes everything.
func (m Mask) Has(flags ...Mask) bool {
	if m&Admin != 0 {
		return true
	}
	for _, f := range flags {
		if m&f != f {
			return false
		}
	}
	return true
}

// HasAny reports whether at least one of the flags is set. Admin passes everything.
func (m Mask) HasAny(flags ...Mask) bool {
	if m&Admin != 0 {
		return true
	}
	return slices.ContainsFunc(flags, func(f Mask) bool { return m&f != 0 })
}

// Stored cleans a mask before it is saved: unknown bits dropped, implied flags added, and never the admin bit
// (being an admin is the role, so nobody can hand out admin through permissions).
func Stored(m Mask) Mask { return Normalize(m) &^ Admin }

// Normalize drops bits that are not flags and switches on the flags that come with another one
// (Manage requests brings auto-approve and the rest, Request brings movies and shows, ...).
func Normalize(m Mask) Mask {
	m &= All
	for _, f := range Catalog {
		if m&f.Value == 0 {
			continue
		}
		for _, name := range f.Implies {
			m |= byName[name]
		}
	}
	return m
}

// Effective is what a person may do: everything for an admin, otherwise their stored mask.
func Effective(isAdmin bool, stored Mask) Mask {
	if isAdmin {
		return All
	}
	return Stored(stored)
}

var byName = func() map[string]Mask {
	out := make(map[string]Mask, len(Catalog))
	for _, f := range Catalog {
		out[f.Name] = f.Value
	}
	return out
}()

// ByName returns the flag called name (the names of Catalog).
func ByName(name string) (Mask, bool) {
	m, ok := byName[name]
	return m, ok
}

// AutoApproves reports whether a request of this type is approved without an admin.
func (m Mask) AutoApproves(mediaType string) bool {
	if m&Admin != 0 || m&(ManageRequests|AutoApprove) != 0 {
		return true
	}
	if mediaType == "movie" {
		return m&AutoApproveMovie != 0
	}
	return m&AutoApproveTV != 0
}

// MayRequest reports whether a request of this type is allowed.
func (m Mask) MayRequest(mediaType string) bool {
	if m&Admin != 0 || m&Request != 0 {
		return true
	}
	if mediaType == "movie" {
		return m&RequestMovie != 0
	}
	return m&RequestTV != 0
}
