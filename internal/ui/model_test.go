package ui

import (
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
