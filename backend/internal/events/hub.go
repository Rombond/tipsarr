// Package events is the in-memory pub/sub behind the SSE stream. Events are a "something
// changed" nudge; REST stays the source of truth.
package events

import (
	"encoding/json"
	"sync"
)

const bufferSize = 500

type Event struct {
	ID        int64           `json:"-"`
	Type      string          `json:"-"`
	OwnerID   string          `json:"-"` // user who should see it ("" = admins only)
	AdminOnly bool            `json:"-"`
	Data      json.RawMessage `json:"data"`
}

type sub struct {
	userID  string
	isAdmin bool
	ch      chan Event
}

type Hub struct {
	mu     sync.Mutex
	nextID int64
	buf    []Event // ring of the most recent events, oldest first
	subs   map[*sub]struct{}
}

func New() *Hub { return &Hub{subs: map[*sub]struct{}{}} }

func visible(e Event, userID string, isAdmin bool) bool {
	if isAdmin {
		return true
	}
	return !e.AdminOnly && e.OwnerID != "" && e.OwnerID == userID
}

// Publish sends an event to the owner and to admins (or to admins only when adminOnly).
func (h *Hub) Publish(typ, ownerID string, data any, adminOnly bool) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	e := Event{ID: h.nextID, Type: typ, OwnerID: ownerID, AdminOnly: adminOnly, Data: raw}
	h.buf = append(h.buf, e)
	if len(h.buf) > bufferSize {
		h.buf = h.buf[len(h.buf)-bufferSize:]
	}
	for s := range h.subs {
		if !visible(e, s.userID, s.isAdmin) {
			continue
		}
		select {
		case s.ch <- e:
		default: // slow consumer: drop; the client refetches after reconnect
		}
	}
}

// Subscribe returns a live channel plus any buffered events newer than lastID that the
// subscriber may see. Call cancel when the connection ends.
func (h *Hub) Subscribe(userID string, isAdmin bool, lastID int64) (live <-chan Event, replay []Event, cancel func()) {
	s := &sub{userID: userID, isAdmin: isAdmin, ch: make(chan Event, 64)}
	h.mu.Lock()
	for _, e := range h.buf {
		if e.ID > lastID && visible(e, userID, isAdmin) {
			replay = append(replay, e)
		}
	}
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s.ch, replay, func() {
		h.mu.Lock()
		delete(h.subs, s)
		h.mu.Unlock()
	}
}
