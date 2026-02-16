package tasks

import "time"

type TaskAliasRow struct {
	TaskID     string
	AliasType  AliasType
	AliasValue string
	Repo       string
	Branch     string
	PRNumber   int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type GroupCount struct {
	Key   string
	Count int
}
