package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/uptrace/bun"
)

const (
	IssueOpen     = "open"
	IssueResolved = "resolved"
)

type Issue struct {
	bun.BaseModel `bun:"table:issues"`

	ID            string `bun:"id,pk"`
	MediaType     string `bun:"media_type"`
	TMDBID        int64  `bun:"tmdb_id"`
	Title         string `bun:"title"`
	PosterPath    string `bun:"poster_path"`
	Kind          string `bun:"kind"`
	SeasonNumber  int    `bun:"season_number"`
	EpisodeNumber int    `bun:"episode_number"`
	Status        string `bun:"status"`
	CreatedBy     string `bun:"created_by"`
	ResolvedBy    string `bun:"resolved_by"`
	CreatedAt     int64  `bun:"created_at"`
	UpdatedAt     int64  `bun:"updated_at"`
}

type IssueComment struct {
	bun.BaseModel `bun:"table:issue_comments"`

	ID        string `bun:"id,pk"`
	IssueID   string `bun:"issue_id"`
	UserID    string `bun:"user_id"`
	Message   string `bun:"message"`
	CreatedAt int64  `bun:"created_at"`
}

type IssueFilter struct {
	CreatedBy  string
	Status     string // "" = any
	TMDBID     int
	MediaType  string
	Take, Skip int
}

// CreateIssue stores the issue and its first message in one transaction.
func (s *Store) CreateIssue(ctx context.Context, i *Issue, message string) error {
	now := time.Now().Unix()
	i.CreatedAt, i.UpdatedAt = now, now
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(i).Exec(ctx); err != nil {
			return err
		}
		c := &IssueComment{ID: NewID(), IssueID: i.ID, UserID: i.CreatedBy, Message: message, CreatedAt: now}
		_, err := tx.NewInsert().Model(c).Exec(ctx)
		return err
	})
}

func (s *Store) GetIssue(ctx context.Context, id string) (*Issue, error) {
	i := new(Issue)
	err := s.DB.NewSelect().Model(i).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return i, err
}

func (s *Store) UpdateIssue(ctx context.Context, i *Issue) error {
	i.UpdatedAt = time.Now().Unix()
	_, err := s.DB.NewUpdate().Model(i).WherePK().Exec(ctx)
	return err
}

func (s *Store) ListIssues(ctx context.Context, f IssueFilter) ([]Issue, int, error) {
	var rows []Issue
	q := s.DB.NewSelect().Model(&rows).Order("created_at DESC")
	if f.CreatedBy != "" {
		q = q.Where("created_by = ?", f.CreatedBy)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.MediaType != "" {
		q = q.Where("media_type = ?", f.MediaType).Where("tmdb_id = ?", f.TMDBID)
	}
	if f.Take > 0 {
		q = q.Limit(int64(f.Take)).Offset(int64(f.Skip))
	}
	total, err := q.ScanAndCount(ctx)
	return rows, int(total), err
}

// IssueCounts returns open/resolved totals (optionally for one reporter).
func (s *Store) IssueCounts(ctx context.Context, createdBy string) (open, resolved int, err error) {
	var rows []struct {
		Status string `bun:"status"`
		N      int    `bun:"n"`
	}
	q := s.DB.NewSelect().Model((*Issue)(nil)).ColumnExpr("status").ColumnExpr("COUNT(*) AS n").Group("status")
	if createdBy != "" {
		q = q.Where("created_by = ?", createdBy)
	}
	if err = q.Scan(ctx, &rows); err != nil {
		return
	}
	for _, r := range rows {
		if r.Status == IssueOpen {
			open = r.N
		} else {
			resolved += r.N
		}
	}
	return
}

func (s *Store) IssueComments(ctx context.Context, issueIDs []string) ([]IssueComment, error) {
	var rows []IssueComment
	if len(issueIDs) == 0 {
		return rows, nil
	}
	return rows, s.DB.NewSelect().Model(&rows).Where("issue_id IN (?)", bun.In(issueIDs)).Order("created_at", "id").Scan(ctx)
}

func (s *Store) AddIssueComment(ctx context.Context, c *IssueComment) error {
	c.CreatedAt = time.Now().Unix()
	if _, err := s.DB.NewInsert().Model(c).Exec(ctx); err != nil {
		return err
	}
	_, err := s.DB.NewUpdate().Model((*Issue)(nil)).Set("updated_at = ?", c.CreatedAt).Where("id = ?", c.IssueID).Exec(ctx)
	return err
}

func (s *Store) DeleteIssue(ctx context.Context, id string) error {
	return s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*IssueComment)(nil)).Where("issue_id = ?", id).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*Issue)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	})
}
