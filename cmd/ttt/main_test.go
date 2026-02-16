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
	nextPaneID    int64
}

func (f *fakeWezTermClient) Spawn(_ context.Context, _ string, _ string) (int64, error) {
	f.spawnCalls++
	return f.nextPaneID, nil
}

func (f *fakeWezTermClient) ActivatePane(_ context.Context, _ int64) error {
	f.activateCalls++
	return nil
}

func (f *fakeWezTermClient) ListPanes(_ context.Context) ([]wezterm.Pane, error) {
	return nil, nil
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
	if _, ok := decoded["aliases"]; !ok {
		t.Fatalf("expected aliases key in dashboard payload: %#v", decoded)
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
