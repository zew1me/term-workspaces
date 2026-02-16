package main

import (
	"bytes"
	"io"
	"os"
	"strings"
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
	return values
}
