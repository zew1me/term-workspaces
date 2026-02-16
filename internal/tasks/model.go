package tasks

import (
	"fmt"
	"strings"
	"time"
)

type AliasType string

const (
	AliasTypePrePR AliasType = "prepr"
	AliasTypePR    AliasType = "pr"
)

type Task struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TaskAlias struct {
	TaskID    string
	Type      AliasType
	Value     string
	Repo      string
	Branch    string
	PRNumber  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NormalizeRepo(repo string) string {
	return strings.ToLower(strings.TrimSpace(repo))
}

func NormalizeBranch(branch string) string {
	return strings.TrimSpace(branch)
}

func PrePRAliasValue(repo, branch string) string {
	return fmt.Sprintf("prepr:%s:%s", NormalizeRepo(repo), NormalizeBranch(branch))
}

func PRAliasValue(repo string, prNumber int) string {
	return fmt.Sprintf("pr:%s#%d", NormalizeRepo(repo), prNumber)
}
