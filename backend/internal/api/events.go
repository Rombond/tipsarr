package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// eventsHandler is the SSE stream (GET /api/v1/events). One stream per open tab, filtered
// server-side: users see their own request events, admins see everything. Events are a
// "something changed" nudge; clients refetch over REST (and always after a reconnect).
func eventsHandler(d Deps) http.HandlerFunc {
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
		var last int64
		if v := r.Header.Get("Last-Event-ID"); v != "" {
			last, _ = strconv.ParseInt(v, 10, 64)
		}
		live, replay, cancel := d.Hub.Subscribe(u.ID, u.Role == "admin", last)
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
