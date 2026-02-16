package tasks

import (
	"errors"
	"testing"
	"time"
)

func TestSQLiteStoreServicePrePRToPRLink(t *testing.T) {
	t.Parallel()

	h := newSQLiteTestHarness(t)

	preTask, created, err := h.Service.GetOrCreatePrePRTask(h.Ctx, "owner/repo", "feature/sqlite")
	if err != nil {
		t.Fatalf("GetOrCreatePrePRTask: %v", err)
	}
	if !created {
		t.Fatalf("expected pre-PR task to be created")
	}

	linkedTask, status, err := h.Service.LinkPRToPrePR(h.Ctx, "owner/repo", "feature/sqlite", 88)
	if err != nil {
		t.Fatalf("LinkPRToPrePR: %v", err)
	}
	if status != LinkStatusLinkedExistingPrePR {
		t.Fatalf("unexpected status: %q", status)
	}
	if linkedTask.ID != preTask.ID {
		t.Fatalf("expected linked task id %q, got %q", preTask.ID, linkedTask.ID)
	}

	alias, found, err := h.Store.GetAlias(h.Ctx, PRAliasValue("owner/repo", 88))
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

	h := newSQLiteTestHarness(t)
	now := time.Now().UTC()

	first := Task{ID: "task_one", CreatedAt: now, UpdatedAt: now}
	second := Task{ID: "task_two", CreatedAt: now, UpdatedAt: now}

	if err := h.Store.CreateTask(h.Ctx, first); err != nil {
		t.Fatalf("CreateTask(first): %v", err)
	}
	if err := h.Store.CreateTask(h.Ctx, second); err != nil {
		t.Fatalf("CreateTask(second): %v", err)
	}

	aliasValue := PRAliasValue("owner/repo", 12)
	if err := h.Store.UpsertAlias(h.Ctx, TaskAlias{
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

	err := h.Store.UpsertAlias(h.Ctx, TaskAlias{
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

func TestSQLiteStoreListTaskAliasRows(t *testing.T) {
	t.Parallel()

	h := newSQLiteTestHarness(t)

	if _, _, err := h.Service.GetOrCreatePrePRTask(h.Ctx, "owner/repo", "feature/list"); err != nil {
		t.Fatalf("GetOrCreatePrePRTask: %v", err)
	}
	if _, _, err := h.Service.LinkPRToPrePR(h.Ctx, "owner/repo", "feature/list", 55); err != nil {
		t.Fatalf("LinkPRToPrePR: %v", err)
	}

	rows, err := h.Store.ListTaskAliasRows(h.Ctx)
	if err != nil {
		t.Fatalf("ListTaskAliasRows: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least two alias rows, got %d", len(rows))
	}

	var hasPrePR, hasPR bool
	for _, row := range rows {
		switch row.AliasType {
		case AliasTypePrePR:
			hasPrePR = true
		case AliasTypePR:
			if row.PRNumber == 55 {
				hasPR = true
			}
		}
	}

	if !hasPrePR {
		t.Fatalf("expected at least one prepr row")
	}
	if !hasPR {
		t.Fatalf("expected PR row for PR 55")
	}
}

func TestSQLiteStoreListTaskAliasGroupCounts(t *testing.T) {
	t.Parallel()

	h := newSQLiteTestHarness(t)

	if _, _, err := h.Service.GetOrCreatePrePRTask(h.Ctx, "owner/repo", "feature/group"); err != nil {
		t.Fatalf("GetOrCreatePrePRTask: %v", err)
	}
	if _, _, err := h.Service.LinkPRToPrePR(h.Ctx, "owner/repo", "feature/group", 19); err != nil {
		t.Fatalf("LinkPRToPrePR: %v", err)
	}

	aliasTypeCounts, err := h.Store.ListTaskAliasGroupCounts(h.Ctx, "alias_type")
	if err != nil {
		t.Fatalf("ListTaskAliasGroupCounts(alias_type): %v", err)
	}
	if len(aliasTypeCounts) < 2 {
		t.Fatalf("expected at least two alias_type groups, got %d", len(aliasTypeCounts))
	}

	repoCounts, err := h.Store.ListTaskAliasGroupCounts(h.Ctx, "repo")
	if err != nil {
		t.Fatalf("ListTaskAliasGroupCounts(repo): %v", err)
	}
	if len(repoCounts) == 0 || repoCounts[0].Key != "owner/repo" {
		t.Fatalf("expected repo group for owner/repo, got %#v", repoCounts)
	}
}
