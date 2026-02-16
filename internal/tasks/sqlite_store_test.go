package tasks

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newSQLiteTestStore(t *testing.T) *SQLiteStore {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	store, err := NewSQLiteStoreFromDB(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("NewSQLiteStoreFromDB: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})

	return store
}

func TestSQLiteStoreServicePrePRToPRLink(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newSQLiteTestStore(t)
	service := NewService(store)

	preTask, created, err := service.GetOrCreatePrePRTask(ctx, "owner/repo", "feature/sqlite")
	if err != nil {
		t.Fatalf("GetOrCreatePrePRTask: %v", err)
	}
	if !created {
		t.Fatalf("expected pre-PR task to be created")
	}

	linkedTask, status, err := service.LinkPRToPrePR(ctx, "owner/repo", "feature/sqlite", 88)
	if err != nil {
		t.Fatalf("LinkPRToPrePR: %v", err)
	}
	if status != LinkStatusLinkedExistingPrePR {
		t.Fatalf("unexpected status: %q", status)
	}
	if linkedTask.ID != preTask.ID {
		t.Fatalf("expected linked task id %q, got %q", preTask.ID, linkedTask.ID)
	}

	alias, found, err := store.GetAlias(ctx, PRAliasValue("owner/repo", 88))
	if err != nil {
		t.Fatalf("GetAlias: %v", err)
	}
	if !found {
		t.Fatalf("expected PR alias to be present")
	}
	if alias.TaskID != preTask.ID {
		t.Fatalf("expected PR alias TaskID %q, got %q", preTask.ID, alias.TaskID)
	}
}

func TestSQLiteStoreRejectsAliasRebind(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newSQLiteTestStore(t)
	now := time.Now().UTC()

	first := Task{ID: "task_one", CreatedAt: now, UpdatedAt: now}
	second := Task{ID: "task_two", CreatedAt: now, UpdatedAt: now}

	if err := store.CreateTask(ctx, first); err != nil {
		t.Fatalf("CreateTask(first): %v", err)
	}
	if err := store.CreateTask(ctx, second); err != nil {
		t.Fatalf("CreateTask(second): %v", err)
	}

	aliasValue := PRAliasValue("owner/repo", 12)
	if err := store.UpsertAlias(ctx, TaskAlias{
		TaskID:    first.ID,
		Type:      AliasTypePR,
		Value:     aliasValue,
		Repo:      "owner/repo",
		PRNumber:  12,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("UpsertAlias(first): %v", err)
	}

	err := store.UpsertAlias(ctx, TaskAlias{
		TaskID:    second.ID,
		Type:      AliasTypePR,
		Value:     aliasValue,
		Repo:      "owner/repo",
		PRNumber:  12,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if !errors.Is(err, ErrAliasAlreadyBound) {
		t.Fatalf("expected ErrAliasAlreadyBound, got %v", err)
	}
}
