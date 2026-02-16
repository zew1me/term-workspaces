package tasks

import (
	"context"
	"errors"
)

var (
	ErrAliasAlreadyBound = errors.New("alias already bound to a different task")
	ErrTaskNotFound      = errors.New("task not found")
)

type Store interface {
	CreateTask(ctx context.Context, task Task) error
	GetTaskByAlias(ctx context.Context, aliasValue string) (Task, bool, error)
	GetAlias(ctx context.Context, aliasValue string) (TaskAlias, bool, error)
	UpsertAlias(ctx context.Context, alias TaskAlias) error
}
