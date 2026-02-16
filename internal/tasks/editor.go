package tasks

import "strings"

func ResolveEditorCommand(editorEnv, notePath string) (string, []string) {
	fields := strings.Fields(strings.TrimSpace(editorEnv))
	if len(fields) == 0 {
		return "open", []string{"-e", notePath}
	}

	name := fields[0]
	args := append(fields[1:], notePath)
	return name, args
}
