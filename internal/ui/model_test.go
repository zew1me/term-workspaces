package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestDummyModelRendersThreeTabs(t *testing.T) {
	model := NewDummyModel()
	view := model.View()
	for _, tab := range []string{"PR Queue", "Open Sessions", "Events"} {
		if !strings.Contains(view, tab) {
			t.Fatalf("expected tab %q in view: %q", tab, view)
		}
	}
}

func TestDummyModelTabSwitch(t *testing.T) {
	model := NewDummyModel()
	updated := model.NextTab()
	if updated.activeTab != 1 {
		t.Fatalf("expected active tab index 1, got %d", updated.activeTab)
	}
}

func TestZeroValueModelDoesNotPanic(t *testing.T) {
	var model Model
	_ = model.NextTab()
	_ = model.PrevTab()
	view := model.View()
	if !strings.Contains(view, "ttt UI") {
		t.Fatalf("expected header in zero-value view: %q", view)
	}
}

func TestViewHandlesOutOfRangeActiveTab(t *testing.T) {
	model := NewDummyModel()
	model.activeTab = 999
	view := model.View()
	if !strings.Contains(view, "[ PR Queue ]") {
		t.Fatalf("expected fallback to first tab: %q", view)
	}
}

func TestRunInteractiveSwitchesTabsAndQuits(t *testing.T) {
	input := strings.NewReader("2\nq\n")
	var output bytes.Buffer

	if err := RunInteractive(NewDummyModel(), input, &output); err != nil {
		t.Fatalf("RunInteractive failed: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "[ Open Sessions ]") {
		t.Fatalf("expected switched tab in output: %q", text)
	}
	if !strings.Contains(text, "command>") {
		t.Fatalf("expected prompt in output: %q", text)
	}
}
