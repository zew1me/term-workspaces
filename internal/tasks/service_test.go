package tasks

import (
	"context"
	"testing"
)

func TestGetOrCreatePrePRTaskIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMemoryStore()
	service := NewService(store)

	first, created, err := service.GetOrCreatePrePRTask(ctx, "Owner/Repo", "feature/x")
	if err != nil {
		t.Fatalf("GetOrCreatePrePRTask first call error: %v", err)
	}
	if !created {
		t.Fatalf("expected first call to create task")
	}

	second, createdAgain, err := service.GetOrCreatePrePRTask(ctx, "owner/repo", "feature/x")
	if err != nil {
		t.Fatalf("GetOrCreatePrePRTask second call error: %v", err)
	}
	if createdAgain {
		t.Fatalf("expected second call to reuse existing task")
	}
	if first.ID != second.ID {
		t.Fatalf("expected same task id, got %q and %q", first.ID, second.ID)
	}

	alias, found, err := store.GetAlias(ctx, PrePRAliasValue("owner/repo", "feature/x"))
	if err != nil {
		t.Fatalf("GetAlias error: %v", err)
	}
	if !found {
		t.Fatalf("expected pre-PR alias to exist")
	}
	if alias.Type != AliasTypePrePR {
		t.Fatalf("expected alias type %q, got %q", AliasTypePrePR, alias.Type)
	}
}

func TestLinkPRToExistingPrePRTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMemoryStore()
	service := NewService(store)

	preTask, created, err := service.GetOrCreatePrePRTask(ctx, "owner/repo", "feature/abc")
	if err != nil {
		t.Fatalf("GetOrCreatePrePRTask error: %v", err)
	}
	if !created {
		t.Fatalf("expected pre-PR task to be created")
	}

	linkedTask, status, err := service.LinkPRToPrePR(ctx, "owner/repo", "feature/abc", 42)
	if err != nil {
		t.Fatalf("LinkPRToPrePR error: %v", err)
	}
	if status != LinkStatusLinkedExistingPrePR {
		t.Fatalf("unexpected link status: %q", status)
	}
	if linkedTask.ID != preTask.ID {
		t.Fatalf("expected PR alias to link to existing pre-PR task")
	}

	prAlias, found, err := store.GetAlias(ctx, PRAliasValue("owner/repo", 42))
	if err != nil {
		t.Fatalf("GetAlias error: %v", err)
	}
	if !found {
		t.Fatalf("expected PR alias to be written")
	}
	if prAlias.TaskID != preTask.ID {
		t.Fatalf("expected PR alias to point to pre-PR task %q, got %q", preTask.ID, prAlias.TaskID)
	}
}

func TestLinkPRCreatesTaskWhenNoPrePRTaskExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMemoryStore()
	service := NewService(store)

	task, status, err := service.LinkPRToPrePR(ctx, "owner/repo", "feature/new", 101)
	if err != nil {
		t.Fatalf("LinkPRToPrePR error: %v", err)
	}
	if status != LinkStatusCreatedFromPR {
		t.Fatalf("expected status %q, got %q", LinkStatusCreatedFromPR, status)
	}
	if task.ID == "" {
		t.Fatalf("expected non-empty task id")
	}

	prAlias, found, err := store.GetAlias(ctx, PRAliasValue("owner/repo", 101))
	if err != nil {
		t.Fatalf("GetAlias error: %v", err)
	}
	if !found {
		t.Fatalf("expected PR alias to exist")
	}
	if prAlias.TaskID != task.ID {
		t.Fatalf("expected PR alias to point to created task %q, got %q", task.ID, prAlias.TaskID)
	}
}

func TestLinkPRReturnsAlreadyLinkedWhenPRAliasExists(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewMemoryStore()
	service := NewService(store)

	task, status, err := service.LinkPRToPrePR(ctx, "owner/repo", "feature/one", 77)
	if err != nil {
		t.Fatalf("first LinkPRToPrePR error: %v", err)
	}
	if status != LinkStatusCreatedFromPR {
		t.Fatalf("expected first status %q, got %q", LinkStatusCreatedFromPR, status)
	}

	secondTask, secondStatus, err := service.LinkPRToPrePR(ctx, "owner/repo", "feature/two", 77)
	if err != nil {
		t.Fatalf("second LinkPRToPrePR error: %v", err)
	}
	if secondStatus != LinkStatusAlreadyLinked {
		t.Fatalf("expected second status %q, got %q", LinkStatusAlreadyLinked, secondStatus)
	}
	if secondTask.ID != task.ID {
		t.Fatalf("expected existing PR task id %q, got %q", task.ID, secondTask.ID)
	}
}
