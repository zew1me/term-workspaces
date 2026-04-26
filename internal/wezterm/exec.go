package wezterm

import (
	"context"
	"fmt"
	"os/exec"
)

func defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	// #nosec G204 -- callers pass fixed wezterm CLI command + arguments.
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v failed: %w (%s)", name, args, err, string(output))
	}
	return output, nil
}
