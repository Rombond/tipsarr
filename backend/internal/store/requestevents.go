package store

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

// RequestEvent is one step of a request's history. Kind is a stage: requested, approved, searching, downloading,
// available, declined or failed.
type RequestEvent struct {
	bun.BaseModel `bun:"table:request_events"`

	ID        string `bun:"id,pk"`
	RequestID string `bun:"request_id"`
	Kind      string `bun:"kind"`
	ActorID   string `bun:"actor_id"` // who did it; empty when the system did
	Note      string `bun:"note"`     // the decline reason or the error, when there is one
	At        int64  `bun:"at"`
}

// AddRequestEvent records a step now.
func (s *Store) AddRequestEvent(ctx context.Context, requestID, kind, actorID, note string) error {
	// the id starts with the time in nanoseconds, so steps of the same second still sort in the order they happened
	id := fmt.Sprintf("%016x", time.Now().UnixNano()) + NewID()[:16]
	e := &RequestEvent{ID: id, RequestID: requestID, Kind: kind, ActorID: actorID, Note: note, At: time.Now().Unix()}
	_, err := s.DB.NewInsert().Model(e).Exec(ctx)
	return err
}

// HasRequestEvent reports whether a request already has a step of this kind.
func (s *Store) HasRequestEvent(ctx context.Context, requestID, kind string) (bool, error) {
	return s.DB.NewSelect().Model((*RequestEvent)(nil)).Where("request_id = ?", requestID).Where("kind = ?", kind).Exists(ctx)
}

// RequestEvents returns the steps of these requests, oldest first.
func (s *Store) RequestEvents(ctx context.Context, ids []string) (map[string][]RequestEvent, error) {
	out := map[string][]RequestEvent{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []RequestEvent
	if err := s.DB.NewSelect().Model(&rows).Where("request_id IN (?)", bun.In(ids)).Order("at", "id").Scan(ctx); err != nil {
		return nil, err
	}
	for _, e := range rows {
		out[e.RequestID] = append(out[e.RequestID], e)
	}
	return out, nil
}
