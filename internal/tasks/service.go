package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type LinkStatus string

const (
	LinkStatusLinkedExistingPrePR LinkStatus = "linked_existing_prepr"
	LinkStatusCreatedFromPR       LinkStatus = "created_from_pr"
	LinkStatusAlreadyLinked       LinkStatus = "already_linked"
)

type Service struct {
	store      Store
	now        func() time.Time
	idSequence atomic.Uint64
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   time.Now,
	}
}

func (s *Service) nextTaskID(now time.Time) string {
	next := s.idSequence.Add(1)
	return fmt.Sprintf("task_%d_%d", now.UnixNano(), next)
}

func (s *Service) GetOrCreatePrePRTask(ctx context.Context, repo, branch string) (Task, bool, error) {
	repo = NormalizeRepo(repo)
	branch = NormalizeBranch(branch)
	aliasValue := PrePRAliasValue(repo, branch)

	task, found, err := s.store.GetTaskByAlias(ctx, aliasValue)
	if err != nil {
		return Task{}, false, err
	}
	if found {
		return task, false, nil
	}

	now := s.now().UTC()
	task = Task{
		ID:        s.nextTaskID(now),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.CreateTask(ctx, task); err != nil {
		return Task{}, false, err
	}

	alias := TaskAlias{
		TaskID:    task.ID,
		Type:      AliasTypePrePR,
		Value:     aliasValue,
		Repo:      repo,
		Branch:    branch,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.UpsertAlias(ctx, alias); err != nil {
		return Task{}, false, err
	}

	return task, true, nil
}

func (s *Service) LinkPRToPrePR(ctx context.Context, repo, branch string, prNumber int) (Task, LinkStatus, error) {
	repo = NormalizeRepo(repo)
	branch = NormalizeBranch(branch)

	prAliasValue := PRAliasValue(repo, prNumber)
	prTask, prFound, err := s.store.GetTaskByAlias(ctx, prAliasValue)
	if err != nil {
		return Task{}, "", err
	}
	if prFound {
		return prTask, LinkStatusAlreadyLinked, nil
	}

	preAliasValue := PrePRAliasValue(repo, branch)
	preTask, preFound, err := s.store.GetTaskByAlias(ctx, preAliasValue)
	if err != nil {
		return Task{}, "", err
	}

	now := s.now().UTC()
	target := preTask
	status := LinkStatusLinkedExistingPrePR

	if !preFound {
		target = Task{
			ID:        s.nextTaskID(now),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.store.CreateTask(ctx, target); err != nil {
			return Task{}, "", err
		}
		status = LinkStatusCreatedFromPR
	}

	prAlias := TaskAlias{
		TaskID:    target.ID,
		Type:      AliasTypePR,
		Value:     prAliasValue,
		Repo:      repo,
		PRNumber:  prNumber,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.UpsertAlias(ctx, prAlias); err != nil {
		if errors.Is(err, ErrAliasAlreadyBound) {
			return Task{}, "", err
		}
		return Task{}, "", err
	}

	return target, status, nil
}

func (s *Service) GetTaskByPrePR(ctx context.Context, repo, branch string) (Task, bool, error) {
	aliasValue := PrePRAliasValue(repo, branch)
	return s.store.GetTaskByAlias(ctx, aliasValue)
}

func (s *Service) GetTaskByPR(ctx context.Context, repo string, prNumber int) (Task, bool, error) {
	aliasValue := PRAliasValue(repo, prNumber)
	return s.store.GetTaskByAlias(ctx, aliasValue)
}
