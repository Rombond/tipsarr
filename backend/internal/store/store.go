// Package store owns all database access. SQLite is the default; PostgreSQL and
// MySQL are selected through the DB URL.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	DB      *bun.DB
	Dialect string // sqlite | postgres | mysql
}

// Open connects to the database described by dbURL and applies pending migrations.
func Open(ctx context.Context, dbURL string) (*Store, error) {
	s, err := connect(dbURL)
	if err != nil {
		return nil, err
	}
	if err := s.migrate(ctx); err != nil {
		_ = s.DB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func connect(dbURL string) (*Store, error) {
	switch {
	case strings.HasPrefix(dbURL, "sqlite:"):
		path := strings.TrimPrefix(strings.TrimPrefix(dbURL, "sqlite:"), "//")
		dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
		sqlDB, err := sql.Open("sqlite", dsn)
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(1) // SQLite: single writer, avoids "database is locked"
		return &Store{DB: bun.NewDB(sqlDB, sqlitedialect.New()), Dialect: "sqlite"}, nil

	case strings.HasPrefix(dbURL, "postgres://"), strings.HasPrefix(dbURL, "postgresql://"):
		sqlDB, err := sql.Open("pgx", dbURL)
		if err != nil {
			return nil, err
		}
		return &Store{DB: bun.NewDB(sqlDB, pgdialect.New()), Dialect: "postgres"}, nil

	case strings.HasPrefix(dbURL, "mysql://"):
		dsn, err := mysqlDSN(dbURL)
		if err != nil {
			return nil, err
		}
		sqlDB, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		return &Store{DB: bun.NewDB(sqlDB, mysqldialect.New()), Dialect: "mysql"}, nil
	}
	return nil, fmt.Errorf("unsupported DB URL %q (use sqlite:<path>, postgres://… or mysql://…)", redact(dbURL))
}

// mysqlDSN converts mysql://user:pass@host:3306/db into the go-sql-driver DSN.
func mysqlDSN(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := u.Host
	if u.Port() == "" {
		host += ":3306"
	}
	pass, _ := u.User.Password()
	return fmt.Sprintf("%s:%s@tcp(%s)%s?parseTime=true&charset=utf8mb4", u.User.Username(), pass, host, u.Path), nil
}

func redact(s string) string {
	if i := strings.Index(s, "://"); i >= 0 {
		return s[:i+3] + "…"
	}
	return s
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version VARCHAR(191) NOT NULL PRIMARY KEY, applied_at BIGINT NOT NULL)`); err != nil {
		return err
	}
	var applied []string
	if err := s.DB.NewSelect().TableExpr("schema_migrations").Column("version").Scan(ctx, &applied); err != nil {
		return err
	}
	done := map[string]bool{}
	for _, v := range applied {
		done[v] = true
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		if done[name] {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		err = s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			for _, stmt := range splitStatements(string(body)) {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, name, time.Now().Unix())
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// splitStatements splits a migration file on ';', dropping "--" comment lines.
func splitStatements(src string) []string {
	var lines []string
	for _, l := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "--") {
			continue
		}
		lines = append(lines, l)
	}
	var out []string
	for _, part := range strings.Split(strings.Join(lines, "\n"), ";") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
