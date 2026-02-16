package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"term-workspaces/internal/wezterm"
	"testing"
)

func TestRunTaskEnsurePrePRCreatedThenExisting(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	out1, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/cli-test",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("first ensure-prepr run failed: %v", err)
	}
	first := parseKVLine(t, out1)
	if got := first["status"]; got != "created" {
		t.Fatalf("expected first status=created, got %q (output=%q)", got, out1)
	}

	out2, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/cli-test",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("second ensure-prepr run failed: %v", err)
	}
	second := parseKVLine(t, out2)
	if got := second["status"]; got != "existing" {
		t.Fatalf("expected second status=existing, got %q (output=%q)", got, out2)
	}
	if first["task_id"] != second["task_id"] {
		t.Fatalf("expected same task_id across ensure-prepr runs, got %q and %q", first["task_id"], second["task_id"])
	}
}

func TestRunUIPreview(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	out, err := captureStdout(func() error {
		return run([]string{"ui", "--preview", "--db", dbPath})
	})
	if err != nil {
		t.Fatalf("ui --preview failed: %v", err)
	}
	if !strings.Contains(out, "ttt UI (preview)") {
		t.Fatalf("expected preview header in output: %q", out)
	}
	if !strings.Contains(out, "PR Queue") {
		t.Fatalf("expected tabs in output: %q", out)
	}
	if !strings.Contains(out, "no task aliases found") {
		t.Fatalf("expected empty-state queue in output: %q", out)
	}
}

func TestRunUIPreviewUsesRealStoreData(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 1200}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/ui-real-data",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("open-session run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"ui", "--preview", "--db", dbPath})
	})
	if err != nil {
		t.Fatalf("ui --preview failed: %v", err)
	}
	if !strings.Contains(out, "prepr:zew1me/term-workspaces:feature/ui-real-data") {
		t.Fatalf("expected real alias in preview output: %q", out)
	}
	if !strings.Contains(out, "session=open(pane=1200)") {
		t.Fatalf("expected open session status in preview output: %q", out)
	}
}

func TestRunTaskLinkPRReusesPrePRTaskID(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	preOut, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/link-test",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}
	pre := parseKVLine(t, preOut)

	linkOut, err := captureStdout(func() error {
		return run([]string{
			"task", "link-pr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/link-test",
			"--pr", "321",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("link-pr run failed: %v", err)
	}
	link := parseKVLine(t, linkOut)
	if got := link["status"]; got != "linked_existing_prepr" {
		t.Fatalf("expected status=linked_existing_prepr, got %q (output=%q)", got, linkOut)
	}
	if pre["task_id"] != link["task_id"] {
		t.Fatalf("expected link-pr to reuse pre-PR task id, got %q and %q", pre["task_id"], link["task_id"])
	}
}

func TestRunTaskEnsureNoteCreatesThenReusesFile(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	notesDir := t.TempDir() + "/notes"

	out1, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-note",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/notes-test",
			"--db", dbPath,
			"--notes-dir", notesDir,
		})
	})
	if err != nil {
		t.Fatalf("first ensure-note run failed: %v", err)
	}
	first := parseKVLine(t, out1)
	if got := first["status"]; got != "created" {
		t.Fatalf("expected first status=created, got %q (output=%q)", got, out1)
	}
	notePath := first["note_path"]
	if notePath == "" {
		t.Fatalf("expected note_path in output: %q", out1)
	}
	// #nosec G304 -- notePath is generated in test setup via t.TempDir.
	content, readErr := os.ReadFile(notePath)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", notePath, readErr)
	}
	if !strings.Contains(string(content), "# Task State") {
		t.Fatalf("expected task note template header, got: %q", string(content))
	}

	out2, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-note",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/notes-test",
			"--db", dbPath,
			"--notes-dir", notesDir,
		})
	})
	if err != nil {
		t.Fatalf("second ensure-note run failed: %v", err)
	}
	second := parseKVLine(t, out2)
	if got := second["status"]; got != "existing" {
		t.Fatalf("expected second status=existing, got %q (output=%q)", got, out2)
	}
	if second["note_path"] != notePath {
		t.Fatalf("expected same note path, got %q and %q", notePath, second["note_path"])
	}
}

func TestRunTaskEnsureNoteViaPRAlias(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	notesDir := t.TempDir() + "/notes"

	preOut, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/pr-note",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}
	pre := parseKVLine(t, preOut)

	_, err = captureStdout(func() error {
		return run([]string{
			"task", "link-pr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/pr-note",
			"--pr", "404",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("link-pr run failed: %v", err)
	}

	noteOut, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-note",
			"--repo", "zew1me/term-workspaces",
			"--pr", "404",
			"--db", dbPath,
			"--notes-dir", notesDir,
		})
	})
	if err != nil {
		t.Fatalf("ensure-note via PR run failed: %v", err)
	}
	note := parseKVLine(t, noteOut)
	if note["task_id"] != pre["task_id"] {
		t.Fatalf("expected note task_id %q to match pre-pr task_id %q", note["task_id"], pre["task_id"])
	}
	if note["note_path"] == "" {
		t.Fatalf("expected note_path in output: %q", noteOut)
	}
}

func TestRunTaskEnsureNoteFailsWhenPRAliasMissing(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	notesDir := t.TempDir() + "/notes"

	err := run([]string{
		"task", "ensure-note",
		"--repo", "zew1me/term-workspaces",
		"--pr", "999",
		"--db", dbPath,
		"--notes-dir", notesDir,
	})
	if err == nil {
		t.Fatalf("expected ensure-note to fail for missing PR alias")
	}
	if !strings.Contains(err.Error(), "no task found for") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTaskOpenNoteDryRunUsesEditorEnv(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	notesDir := t.TempDir() + "/notes"
	t.Setenv("EDITOR", "vim -u NONE")

	out, err := captureStdout(func() error {
		return run([]string{
			"task", "open-note",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/open-note",
			"--db", dbPath,
			"--notes-dir", notesDir,
			"--dry-run",
		})
	})
	if err != nil {
		t.Fatalf("open-note dry-run failed: %v", err)
	}
	fields := parseKVLine(t, out)
	if fields["editor"] != "vim" {
		t.Fatalf("expected editor=vim in output, got %q (%q)", fields["editor"], out)
	}
	if !strings.Contains(out, "NONE") {
		t.Fatalf("expected dry-run output args to include 'NONE': %q", out)
	}
}

func TestRunTaskListIncludesCreatedAliases(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/list-test",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "link-pr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/list-test",
			"--pr", "987",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("link-pr run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"task", "list", "--db", dbPath})
	})
	if err != nil {
		t.Fatalf("task list run failed: %v", err)
	}
	if !strings.Contains(out, "task_id\talias_type\talias_value\trepo\tbranch\tpr") {
		t.Fatalf("expected list header in output: %q", out)
	}
	if !strings.Contains(out, "prepr:zew1me/term-workspaces:feature/list-test") {
		t.Fatalf("expected prepr alias in output: %q", out)
	}
	if !strings.Contains(out, "pr:zew1me/term-workspaces#987") {
		t.Fatalf("expected pr alias in output: %q", out)
	}
}

func TestRunTaskListGroupByAliasType(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/group-list",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "link-pr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/group-list",
			"--pr", "765",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("link-pr run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"task", "list", "--db", dbPath, "--group-by", "alias_type"})
	})
	if err != nil {
		t.Fatalf("task list --group-by alias_type failed: %v", err)
	}
	if !strings.Contains(out, "group_key\tcount") {
		t.Fatalf("expected grouped header in output: %q", out)
	}
	if !strings.Contains(out, "prepr") || !strings.Contains(out, "pr") {
		t.Fatalf("expected grouped output to include prepr and pr: %q", out)
	}
}

func TestRunTaskListJSON(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/json-list",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"task", "list", "--db", dbPath, "--json"})
	})
	if err != nil {
		t.Fatalf("task list --json run failed: %v", err)
	}

	type taskRow struct {
		TaskID     string `json:"task_id"`
		AliasValue string `json:"alias_value"`
	}
	var rowsTyped []taskRow
	if err := json.Unmarshal([]byte(out), &rowsTyped); err != nil {
		t.Fatalf("json.Unmarshal failed: %v (output=%q)", err, out)
	}
	if len(rowsTyped) == 0 {
		t.Fatalf("expected at least one row in json output")
	}
	if rowsTyped[0].AliasValue == "" || rowsTyped[0].TaskID == "" {
		t.Fatalf("expected alias_value/task_id in first row: %#v", rowsTyped[0])
	}
}

func TestRunTaskListGroupByAliasTypeJSON(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "ensure-prepr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/json-group",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("ensure-prepr run failed: %v", err)
	}
	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "link-pr",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/json-group",
			"--pr", "111",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("link-pr run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"task", "list", "--db", dbPath, "--group-by", "alias_type", "--json"})
	})
	if err != nil {
		t.Fatalf("task list grouped json run failed: %v", err)
	}

	type groupRow struct {
		Key   string `json:"key"`
		Count int    `json:"count"`
	}
	var groups []groupRow
	if err := json.Unmarshal([]byte(out), &groups); err != nil {
		t.Fatalf("json.Unmarshal failed: %v (output=%q)", err, out)
	}
	if len(groups) == 0 {
		t.Fatalf("expected grouped json rows")
	}
	if groups[0].Key == "" || groups[0].Count <= 0 {
		t.Fatalf("expected key/count in grouped row: %#v", groups[0])
	}
}

type fakeWezTermClient struct {
	spawnCalls    int
	activateCalls int
	killCalls     int
	nextPaneID    int64
	spawned       []wezterm.Pane
	panes         []wezterm.Pane
	listErr       error
	activateErr   error
	killErr       error
}

func (f *fakeWezTermClient) Spawn(_ context.Context, workspace string, _ string) (int64, error) {
	f.spawnCalls++
	paneID := f.nextPaneID + int64(f.spawnCalls-1)
	pane := wezterm.Pane{PaneID: paneID, Workspace: workspace}
	f.spawned = append(f.spawned, pane)
	f.panes = append(f.panes, pane)
	return paneID, nil
}

func (f *fakeWezTermClient) ActivatePane(_ context.Context, _ int64) error {
	f.activateCalls++
	if f.activateErr != nil {
		return f.activateErr
	}
	return nil
}

func (f *fakeWezTermClient) KillPane(_ context.Context, paneID int64) error {
	f.killCalls++
	if f.killErr != nil {
		return f.killErr
	}
	filtered := make([]wezterm.Pane, 0, len(f.panes))
	for _, pane := range f.panes {
		if pane.PaneID == paneID {
			continue
		}
		filtered = append(filtered, pane)
	}
	f.panes = filtered
	return nil
}

func (f *fakeWezTermClient) ListPanes(_ context.Context) ([]wezterm.Pane, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]wezterm.Pane(nil), f.panes...), nil
}

func TestRunTaskOpenSessionSpawnThenActivate(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 7001}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	out1, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/session-open",
			"--db", dbPath,
			"--cwd", "/tmp/work",
		})
	})
	if err != nil {
		t.Fatalf("first open-session run failed: %v", err)
	}
	first := parseKVLine(t, out1)
	if first["status"] != "spawned" {
		t.Fatalf("expected first status=spawned, got %q (%q)", first["status"], out1)
	}

	out2, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/session-open",
			"--db", dbPath,
			"--cwd", "/tmp/work",
		})
	})
	if err != nil {
		t.Fatalf("second open-session run failed: %v", err)
	}
	second := parseKVLine(t, out2)
	if second["status"] != "activated" {
		t.Fatalf("expected second status=activated, got %q (%q)", second["status"], out2)
	}
	if first["task_id"] != second["task_id"] {
		t.Fatalf("expected same task_id across open-session runs")
	}
	if fake.spawnCalls != 1 {
		t.Fatalf("expected one spawn call, got %d", fake.spawnCalls)
	}
	if fake.activateCalls != 1 {
		t.Fatalf("expected one activate call, got %d", fake.activateCalls)
	}
}

func TestRunTaskOpenSessionRespawnsWhenStoredPaneMissing(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 8001}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	out1, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/session-stale",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("first open-session run failed: %v", err)
	}
	first := parseKVLine(t, out1)
	if first["status"] != "spawned" {
		t.Fatalf("expected first status=spawned, got %q (%q)", first["status"], out1)
	}

	// Simulate external pane close so the stored pane id becomes stale.
	fake.panes = nil

	out2, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/session-stale",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("second open-session run failed: %v", err)
	}
	second := parseKVLine(t, out2)
	if second["status"] != "spawned" {
		t.Fatalf("expected second status=spawned after stale reconcile, got %q (%q)", second["status"], out2)
	}
	if first["workspace"] != second["workspace"] {
		t.Fatalf("expected stable workspace across repeated opens, got %q and %q", first["workspace"], second["workspace"])
	}
	if fake.spawnCalls != 2 {
		t.Fatalf("expected two spawn calls, got %d", fake.spawnCalls)
	}
	if fake.activateCalls != 0 {
		t.Fatalf("expected zero activate calls for stale session, got %d", fake.activateCalls)
	}
}

func TestRunTaskSessionsJSONAndGroupedStatus(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 42}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/session-list",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("open-session run failed: %v", err)
	}

	jsonOut, err := captureStdout(func() error {
		return run([]string{"task", "sessions", "--db", dbPath, "--json"})
	})
	if err != nil {
		t.Fatalf("task sessions --json failed: %v", err)
	}
	var sessions []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &sessions); err != nil {
		t.Fatalf("json.Unmarshal sessions failed: %v (%q)", err, jsonOut)
	}
	if len(sessions) == 0 {
		t.Fatalf("expected at least one session in json output")
	}

	groupOut, err := captureStdout(func() error {
		return run([]string{"task", "sessions", "--db", dbPath, "--group-by", "status", "--json"})
	})
	if err != nil {
		t.Fatalf("task sessions grouped --json failed: %v", err)
	}
	var groups []map[string]any
	if err := json.Unmarshal([]byte(groupOut), &groups); err != nil {
		t.Fatalf("json.Unmarshal grouped sessions failed: %v (%q)", err, groupOut)
	}
	if len(groups) == 0 {
		t.Fatalf("expected session status groups")
	}
}

func TestRunTaskDashboardJSON(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 88}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/dashboard",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("open-session run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{"task", "dashboard", "--db", dbPath, "--json"})
	})
	if err != nil {
		t.Fatalf("task dashboard run failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v (%q)", err, out)
	}
	if _, ok := decoded["groups"]; !ok {
		t.Fatalf("expected groups key in dashboard payload: %#v", decoded)
	}
	if _, ok := decoded["sessions"]; !ok {
		t.Fatalf("expected sessions key in dashboard payload: %#v", decoded)
	}
	if _, ok := decoded["open_sessions"]; !ok {
		t.Fatalf("expected open_sessions key in dashboard payload: %#v", decoded)
	}
	if _, ok := decoded["aliases"]; !ok {
		t.Fatalf("expected aliases key in dashboard payload: %#v", decoded)
	}
	if _, ok := decoded["tasks"]; !ok {
		t.Fatalf("expected tasks key in dashboard payload: %#v", decoded)
	}
}

func TestRunTaskCloseSessionClosesAndClearsPane(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 300}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/close-session",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("open-session run failed: %v", err)
	}

	out, err := captureStdout(func() error {
		return run([]string{
			"task", "close-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/close-session",
			"--db", dbPath,
		})
	})
	if err != nil {
		t.Fatalf("close-session run failed: %v", err)
	}
	fields := parseKVLine(t, out)
	if fields["status"] != "closed" {
		t.Fatalf("expected status=closed, got %q (%q)", fields["status"], out)
	}
	if fake.killCalls != 1 {
		t.Fatalf("expected one kill call, got %d", fake.killCalls)
	}

	sessionsOut, err := captureStdout(func() error {
		return run([]string{"task", "sessions", "--db", dbPath, "--json"})
	})
	if err != nil {
		t.Fatalf("task sessions --json failed: %v", err)
	}
	var sessions []map[string]any
	if err := json.Unmarshal([]byte(sessionsOut), &sessions); err != nil {
		t.Fatalf("json.Unmarshal sessions failed: %v (%q)", err, sessionsOut)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one session row, got %d", len(sessions))
	}
	if sessions[0]["status"] != "closed" {
		t.Fatalf("expected persisted session status=closed, got %#v", sessions[0])
	}
	if pane, ok := sessions[0]["pane_id"].(float64); !ok || pane != 0 {
		t.Fatalf("expected pane_id=0 after close, got %#v", sessions[0]["pane_id"])
	}
}

func TestRunTaskSessionsReconcileUpdatesSessionStatus(t *testing.T) {
	dbPath := t.TempDir() + "/state.db"
	fake := &fakeWezTermClient{nextPaneID: 900}

	originalFactory := newWezTermClient
	newWezTermClient = func() wezterm.Client { return fake }
	t.Cleanup(func() { newWezTermClient = originalFactory })

	if _, err := captureStdout(func() error {
		return run([]string{
			"task", "open-session",
			"--repo", "zew1me/term-workspaces",
			"--branch", "feature/reconcile",
			"--db", dbPath,
		})
	}); err != nil {
		t.Fatalf("open-session run failed: %v", err)
	}

	// Simulate pane no longer alive before reconcile.
	fake.panes = nil

	out, err := captureStdout(func() error {
		return run([]string{"task", "sessions", "--db", dbPath, "--reconcile", "--json"})
	})
	if err != nil {
		t.Fatalf("task sessions --reconcile --json failed: %v", err)
	}
	var sessions []map[string]any
	if err := json.Unmarshal([]byte(out), &sessions); err != nil {
		t.Fatalf("json.Unmarshal sessions failed: %v (%q)", err, out)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one session row, got %d", len(sessions))
	}
	if sessions[0]["status"] != "closed" {
		t.Fatalf("expected reconciled status=closed, got %#v", sessions[0])
	}
}

func TestWorkspaceForTaskIDDeterministic(t *testing.T) {
	taskID := "task_1700000000000_1"
	first := workspaceForTaskID(taskID)
	second := workspaceForTaskID(taskID)
	if first != second {
		t.Fatalf("expected stable workspace, got %q and %q", first, second)
	}
}

func captureStdout(fn func() error) (string, error) {
	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = writer
	runErr := fn()
	_ = writer.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, reader)
	_ = reader.Close()

	return strings.TrimSpace(buf.String()), runErr
}

func parseKVLine(t *testing.T, line string) map[string]string {
	t.Helper()

	values := map[string]string{}
	for _, part := range strings.Fields(line) {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values[key] = value
	}

	if values["task_id"] == "" {
		t.Fatalf("missing task_id in output: %q", line)
	}
	for _, key := range []string{"status"} {
		if values[key] == "" {
			t.Fatalf("missing %s in output: %s", key, fmt.Sprintf("%q", line))
		}
	}
	return values
}
