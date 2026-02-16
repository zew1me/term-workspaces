package tasks

import "testing"

func TestResolveEditorCommandFallback(t *testing.T) {
	name, args := ResolveEditorCommand("", "/tmp/task.md")
	if name != "open" {
		t.Fatalf("expected fallback editor command 'open', got %q", name)
	}
	if len(args) != 2 || args[0] != "-e" || args[1] != "/tmp/task.md" {
		t.Fatalf("unexpected fallback args: %#v", args)
	}
}

func TestResolveEditorCommandFromEnv(t *testing.T) {
	name, args := ResolveEditorCommand("nvim -u NONE", "/tmp/task.md")
	if name != "nvim" {
		t.Fatalf("expected editor command 'nvim', got %q", name)
	}
	if len(args) != 3 || args[0] != "-u" || args[1] != "NONE" || args[2] != "/tmp/task.md" {
		t.Fatalf("unexpected editor args: %#v", args)
	}
}
