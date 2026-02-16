package tasks

import "time"

type SessionStatus string

const (
	SessionStatusOpen    SessionStatus = "open"
	SessionStatusClosed  SessionStatus = "closed"
	SessionStatusUnknown SessionStatus = "unknown"
)

type TaskSession struct {
	TaskID         string        `json:"task_id"`
	Workspace      string        `json:"workspace"`
	PaneID         int64         `json:"pane_id"`
	Cwd            string        `json:"cwd"`
	Command        string        `json:"command"`
	Status         SessionStatus `json:"status"`
	CodexSessionID string        `json:"codex_session_id"`
	LastSeenAt     time.Time     `json:"last_seen_at"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
