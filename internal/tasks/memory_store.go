package tasks

import (
	"context"
	"sync"
)

type MemoryStore struct {
	mu          sync.RWMutex
	tasksByID   map[string]Task
	aliasToTask map[string]string
	aliases     map[string]TaskAlias
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasksByID:   make(map[string]Task),
		aliasToTask: make(map[string]string),
		aliases:     make(map[string]TaskAlias),
	}
}

func (s *MemoryStore) CreateTask(_ context.Context, task Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasksByID[task.ID] = task
	return nil
}

func (s *MemoryStore) GetTask(_ context.Context, taskID string) (Task, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasksByID[taskID]
	return task, ok, nil
}

func (s *MemoryStore) GetTaskByAlias(_ context.Context, aliasValue string) (Task, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	taskID, ok := s.aliasToTask[aliasValue]
	if !ok {
		return Task{}, false, nil
	}
	task, taskOK := s.tasksByID[taskID]
	if !taskOK {
		return Task{}, false, ErrTaskNotFound
	}
	return task, true, nil
}

func (s *MemoryStore) GetAlias(_ context.Context, aliasValue string) (TaskAlias, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alias, ok := s.aliases[aliasValue]
	return alias, ok, nil
}

func (s *MemoryStore) UpsertAlias(_ context.Context, alias TaskAlias) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingTaskID, exists := s.aliasToTask[alias.Value]; exists && existingTaskID != alias.TaskID {
		return ErrAliasAlreadyBound
	}

	if _, taskExists := s.tasksByID[alias.TaskID]; !taskExists {
		return ErrTaskNotFound
	}

	s.aliasToTask[alias.Value] = alias.TaskID
	s.aliases[alias.Value] = alias
	return nil
}
