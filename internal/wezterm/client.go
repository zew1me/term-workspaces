package wezterm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Pane struct {
	PaneID    int64  `json:"pane_id"`
	Workspace string `json:"workspace"`
}

type Client interface {
	Spawn(ctx context.Context, workspace, cwd string) (int64, error)
	ActivatePane(ctx context.Context, paneID int64) error
	ListPanes(ctx context.Context) ([]Pane, error)
}

type ExecFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

type CLIClient struct {
	exec ExecFunc
}

func NewCLIClient() *CLIClient {
	return &CLIClient{exec: defaultExec}
}

func NewCLIClientWithExec(execFn ExecFunc) *CLIClient {
	return &CLIClient{exec: execFn}
}

func (c *CLIClient) Spawn(ctx context.Context, workspace, cwd string) (int64, error) {
	args := []string{"cli", "spawn", "--new-window", "--workspace", workspace}
	if strings.TrimSpace(cwd) != "" {
		args = append(args, "--cwd", cwd)
	}
	output, err := c.exec(ctx, "wezterm", args...)
	if err != nil {
		return 0, fmt.Errorf("wezterm spawn: %w", err)
	}

	paneIDRaw := strings.TrimSpace(string(output))
	paneID, err := strconv.ParseInt(paneIDRaw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse spawn pane id %q: %w", paneIDRaw, err)
	}
	return paneID, nil
}

func (c *CLIClient) ActivatePane(ctx context.Context, paneID int64) error {
	_, err := c.exec(ctx, "wezterm", "cli", "activate-pane", "--pane-id", strconv.FormatInt(paneID, 10))
	if err != nil {
		return fmt.Errorf("wezterm activate-pane %d: %w", paneID, err)
	}
	return nil
}

func (c *CLIClient) ListPanes(ctx context.Context) ([]Pane, error) {
	output, err := c.exec(ctx, "wezterm", "cli", "list", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("wezterm list: %w", err)
	}
	return parseListPanesJSON(output)
}

func parseListPanesJSON(raw []byte) ([]Pane, error) {
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, fmt.Errorf("decode wezterm list json: %w", err)
	}

	entries := make([]Pane, 0)
	walkPanes(generic, "", &entries)
	return dedupePanes(entries), nil
}

func walkPanes(node any, inheritedWorkspace string, out *[]Pane) {
	switch typed := node.(type) {
	case map[string]any:
		workspace := inheritedWorkspace
		if value, ok := typed["workspace"].(string); ok && strings.TrimSpace(value) != "" {
			workspace = value
		}
		if paneID, ok := extractPaneID(typed); ok {
			*out = append(*out, Pane{PaneID: paneID, Workspace: workspace})
		}
		for _, value := range typed {
			walkPanes(value, workspace, out)
		}
	case []any:
		for _, value := range typed {
			walkPanes(value, inheritedWorkspace, out)
		}
	}
}

func extractPaneID(node map[string]any) (int64, bool) {
	value, ok := node["pane_id"]
	if !ok {
		return 0, false
	}

	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func dedupePanes(entries []Pane) []Pane {
	seen := map[int64]Pane{}
	for _, entry := range entries {
		if existing, ok := seen[entry.PaneID]; ok {
			if existing.Workspace == "" && entry.Workspace != "" {
				seen[entry.PaneID] = entry
			}
			continue
		}
		seen[entry.PaneID] = entry
	}

	result := make([]Pane, 0, len(seen))
	for _, pane := range seen {
		result = append(result, pane)
	}
	return result
}
