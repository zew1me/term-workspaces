package wezterm

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestSpawnParsesPaneID(t *testing.T) {
	t.Parallel()

	client := NewCLIClientWithExec(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "wezterm" {
			t.Fatalf("unexpected command name: %q", name)
		}
		expected := []string{"cli", "spawn", "--new-window", "--workspace", "task-1", "--cwd", "/tmp/work"}
		if !reflect.DeepEqual(args, expected) {
			t.Fatalf("unexpected args: %#v", args)
		}
		return []byte("123\n"), nil
	})

	paneID, err := client.Spawn(context.Background(), "task-1", "/tmp/work")
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if paneID != 123 {
		t.Fatalf("expected paneID=123, got %d", paneID)
	}
}

func TestActivatePane(t *testing.T) {
	t.Parallel()

	called := false
	client := NewCLIClientWithExec(func(_ context.Context, name string, args ...string) ([]byte, error) {
		called = true
		expected := []string{"cli", "activate-pane", "--pane-id", "91"}
		if name != "wezterm" || !reflect.DeepEqual(args, expected) {
			t.Fatalf("unexpected call %q %#v", name, args)
		}
		return nil, nil
	})

	if err := client.ActivatePane(context.Background(), 91); err != nil {
		t.Fatalf("ActivatePane returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected activate call")
	}
}

func TestKillPane(t *testing.T) {
	t.Parallel()

	called := false
	client := NewCLIClientWithExec(func(_ context.Context, name string, args ...string) ([]byte, error) {
		called = true
		expected := []string{"cli", "kill-pane", "--pane-id", "91"}
		if name != "wezterm" || !reflect.DeepEqual(args, expected) {
			t.Fatalf("unexpected call %q %#v", name, args)
		}
		return nil, nil
	})

	if err := client.KillPane(context.Background(), 91); err != nil {
		t.Fatalf("KillPane returned error: %v", err)
	}
	if !called {
		t.Fatalf("expected kill call")
	}
}

func TestListPanesParsesWorkspaceHierarchy(t *testing.T) {
	t.Parallel()

	jsonOut := `[
	  {
	    "workspace": "alpha",
	    "tabs": [{"panes": [{"pane_id": 1}, {"pane_id": 2}]}]
	  },
	  {
	    "workspace": "beta",
	    "tabs": [{"panes": [{"pane_id": 3}]}]
	  }
	]`
	client := NewCLIClientWithExec(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(jsonOut), nil
	})

	panes, err := client.ListPanes(context.Background())
	if err != nil {
		t.Fatalf("ListPanes returned error: %v", err)
	}
	if len(panes) != 3 {
		t.Fatalf("expected 3 panes, got %d (%#v)", len(panes), panes)
	}
}

func TestListPanesReturnsErrorOnBadJSON(t *testing.T) {
	t.Parallel()

	client := NewCLIClientWithExec(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("{not-json"), nil
	})
	if _, err := client.ListPanes(context.Background()); err == nil {
		t.Fatalf("expected JSON parse error")
	}
}

func TestSpawnReturnsErrorOnExecFailure(t *testing.T) {
	t.Parallel()

	client := NewCLIClientWithExec(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if _, err := client.Spawn(context.Background(), "task-x", ""); err == nil {
		t.Fatalf("expected spawn error")
	}
}
