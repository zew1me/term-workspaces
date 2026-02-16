package tasks

import (
	"fmt"
	"os"
	"path/filepath"
)

const noteTemplate = `# Task State

## Current Objective

## Status

## Next Actions

## Blockers

## Session Context
`

func NotePath(notesDir, taskID string) string {
	return filepath.Join(notesDir, taskID+".md")
}

func EnsureTaskNote(notesDir, taskID string) (string, bool, error) {
	if taskID == "" {
		return "", false, fmt.Errorf("taskID is required")
	}

	if err := os.MkdirAll(notesDir, 0o750); err != nil {
		return "", false, fmt.Errorf("create notes dir: %w", err)
	}

	path := NotePath(notesDir, taskID)
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("stat note file: %w", err)
	}

	if err := os.WriteFile(path, []byte(noteTemplate), 0o600); err != nil {
		return "", false, fmt.Errorf("write note template: %w", err)
	}
	return path, true, nil
}
