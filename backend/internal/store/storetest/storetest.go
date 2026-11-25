// Package storetest opens throwaway SQLite stores for tests. Migrating a fresh database costs most of a small
// test's time (all migrations, slower still under -race), so the first call migrates one database, keeps its
// bytes, and every call copies them into the test's temporary folder: Open then finds every migration applied.
package storetest

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Rombond/tipsarr/backend/internal/store"
)

var (
	once     sync.Once
	template []byte
	tmplErr  error
)

func build() {
	dir, err := os.MkdirTemp("", "tipsarr-storetest-")
	if err != nil {
		tmplErr = err
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "t.db")
	st, err := store.Open(context.Background(), "sqlite:"+path)
	if err != nil {
		tmplErr = err
		return
	}
	// Closing the last connection checkpoints the WAL into the main file, so the file alone holds the schema.
	if tmplErr = st.Close(); tmplErr != nil {
		return
	}
	template, tmplErr = os.ReadFile(path)
}

// Open returns a migrated, empty SQLite store in t's temporary folder, closed when the test ends.
func Open(t testing.TB) *store.Store {
	t.Helper()
	once.Do(build)
	if tmplErr != nil {
		t.Fatalf("storetest template: %v", tmplErr)
	}
	path := filepath.Join(t.TempDir(), "t.db")
	if err := os.WriteFile(path, template, 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(context.Background(), "sqlite:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
