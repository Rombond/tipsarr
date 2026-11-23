package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Rombond/tipsarr/backend/internal/perms"
)

// eventsHandler is the SSE stream (GET /api/v1/events). One stream per open tab, filtered
// server-side: users see their own request events, admins see everything. Events are a
// "something changed" nudge; clients refetch over REST (and always after a reconnect).
func eventsHandler(d Deps) http.HandlerFunc {
	streams := &streamCounter{n: map[string]int{}}
	return func(w http.ResponseWriter, r *http.Request) {
		u := userFrom(r.Context())
		if u == nil {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		if !streams.acquire(u.ID) {
			http.Error(w, "too many open event streams", http.StatusTooManyRequests)
			return
		}
		defer streams.release(u.ID)
		// a stream lives longer than the server's WriteTimeout: no write deadline for it
		_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		var last int64
		if v := r.Header.Get("Last-Event-ID"); v != "" {
			last, _ = strconv.ParseInt(v, 10, 64)
		}
		live, replay, cancel := d.Hub.Subscribe(u.ID, u.CanAny(perms.RequestView, perms.ManageSettings), last) // everyone's request events, and job status
		defer cancel()

		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")
		h.Set("X-Accel-Buffering", "no") // nginx: do not buffer the stream
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "retry: 3000\n\n")
		flusher.Flush()

		write := func(id int64, typ string, data []byte) bool {
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", id, typ, data); err != nil {
				return false
			}
			flusher.Flush()
			return true
		}
		for _, e := range replay {
			if !write(e.ID, e.Type, e.Data) {
				return
			}
		}
		ping := time.NewTicker(20 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case e := <-live:
				if !write(e.ID, e.Type, e.Data) {
					return
				}
			case <-ping.C:
				if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

const maxStreamsPerUser = 6

// streamCounter counts the open event streams per user of one server (one per eventsHandler, so two servers in
// one process, as in parallel tests, do not share a cap).
type streamCounter struct {
	mu sync.Mutex
	n  map[string]int
}

func (c *streamCounter) acquire(userID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n[userID] >= maxStreamsPerUser {
		return false
	}
	c.n[userID]++
	return true
}

func (c *streamCounter) release(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n[userID] <= 1 {
		delete(c.n, userID)
		return
	}
	c.n[userID]--
}
