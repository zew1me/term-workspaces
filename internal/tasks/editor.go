package tasks

import (
	"path/filepath"
	"strings"
)

var allowedEditors = map[string]struct{}{
	"nvim": {},
	"vim":  {},
	"nano": {},
}

func ResolveEditorCommand(editorEnv, notePath string) (string, []string) {
	fields := strings.Fields(strings.TrimSpace(editorEnv))
	if len(fields) == 0 {
		return "open", []string{"-e", notePath}
	}

	name := filepath.Base(fields[0])
	if _, ok := allowedEditors[name]; !ok {
		return "open", []string{"-e", notePath}
	}

	args := append(fields[1:], notePath)
	return name, args
}
