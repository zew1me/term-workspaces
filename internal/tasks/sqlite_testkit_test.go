package tasks

import (
	"context"
	"testing"
)

type sqliteTestHarness struct {
	Ctx     context.Context
	Store   *SQLiteStore
	Service *Service
}

func newSQLiteTestHarness(t *testing.T) *sqliteTestHarness {
	t.Helper()

	dbPath := t.TempDir() + "/tasks.db"
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	return &sqliteTestHarness{
		Ctx:     context.Background(),
		Store:   store,
		Service: NewService(store),
	}
}
