package tasks

import (
	"os"
	"strings"
	"testing"
)

func TestEnsureTaskNoteCreatesThenReuses(t *testing.T) {
	notesDir := t.TempDir()
	taskID := "task_abc"

	path, created, err := EnsureTaskNote(notesDir, taskID)
	if err != nil {
		t.Fatalf("EnsureTaskNote first call error: %v", err)
	}
	if !created {
		t.Fatalf("expected first call to create note")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "# Task State") {
		t.Fatalf("expected template header in note file")
	}

	pathAgain, createdAgain, err := EnsureTaskNote(notesDir, taskID)
	if err != nil {
		t.Fatalf("EnsureTaskNote second call error: %v", err)
	}
	if createdAgain {
		t.Fatalf("expected second call to reuse existing note")
	}
	if path != pathAgain {
		t.Fatalf("expected same note path, got %q and %q", path, pathAgain)
	}
}

func TestEnsureTaskNoteRequiresTaskID(t *testing.T) {
	_, _, err := EnsureTaskNote(t.TempDir(), "")
	if err == nil {
		t.Fatalf("expected error for empty task id")
	}
}
