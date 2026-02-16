package tasks

import "time"

type TaskAliasRow struct {
	TaskID     string    `json:"task_id"`
	AliasType  AliasType `json:"alias_type"`
	AliasValue string    `json:"alias_value"`
	Repo       string    `json:"repo"`
	Branch     string    `json:"branch"`
	PRNumber   int       `json:"pr_number"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GroupCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}
