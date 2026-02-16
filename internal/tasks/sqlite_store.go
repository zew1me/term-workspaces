package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sqlitegorm "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type SQLiteStore struct {
	db *gorm.DB
}

type sqliteTaskModel struct {
	TaskID    string `gorm:"column:task_id;primaryKey"`
	CreatedAt string `gorm:"column:created_at;not null"`
	UpdatedAt string `gorm:"column:updated_at;not null"`
}

func (sqliteTaskModel) TableName() string { return "tasks" }

type sqliteTaskAliasModel struct {
	AliasValue string `gorm:"column:alias_value;primaryKey"`
	TaskID     string `gorm:"column:task_id;not null;index"`
	AliasType  string `gorm:"column:alias_type;not null"`
	Repo       string `gorm:"column:repo"`
	Branch     string `gorm:"column:branch"`
	PRNumber   *int   `gorm:"column:pr_number"`
	CreatedAt  string `gorm:"column:created_at;not null"`
	UpdatedAt  string `gorm:"column:updated_at;not null"`
}

func (sqliteTaskAliasModel) TableName() string { return "task_aliases" }

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite parent dir: %w", err)
	}

	db, err := gorm.Open(sqlitegorm.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("get sql db handle: %w", err)
	}
	return sqlDB.Close()
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	statements := []string{
		"PRAGMA foreign_keys = ON;",
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS task_aliases (
			alias_value TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			alias_type TEXT NOT NULL,
			repo TEXT,
			branch TEXT,
			pr_number INTEGER,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(task_id) ON DELETE CASCADE
		);`,
	}

	for _, statement := range statements {
		if err := s.db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("run sqlite migration statement: %w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) CreateTask(ctx context.Context, task Task) error {
	model := sqliteTaskModel{
		TaskID:    task.ID,
		CreatedAt: formatTime(task.CreatedAt),
		UpdatedAt: formatTime(task.UpdatedAt),
	}
	if err := s.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetTaskByAlias(ctx context.Context, aliasValue string) (Task, bool, error) {
	var alias sqliteTaskAliasModel
	if err := s.db.WithContext(ctx).Where("alias_value = ?", aliasValue).First(&alias).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Task{}, false, nil
		}
		return Task{}, false, fmt.Errorf("query task alias: %w", err)
	}

	var task sqliteTaskModel
	if err := s.db.WithContext(ctx).Where("task_id = ?", alias.TaskID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Task{}, false, ErrTaskNotFound
		}
		return Task{}, false, fmt.Errorf("query task by alias: %w", err)
	}

	return fromTaskModel(task), true, nil
}

func (s *SQLiteStore) GetAlias(ctx context.Context, aliasValue string) (TaskAlias, bool, error) {
	var model sqliteTaskAliasModel
	if err := s.db.WithContext(ctx).Where("alias_value = ?", aliasValue).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TaskAlias{}, false, nil
		}
		return TaskAlias{}, false, fmt.Errorf("query alias: %w", err)
	}
	return fromAliasModel(model), true, nil
}

func (s *SQLiteStore) UpsertAlias(ctx context.Context, alias TaskAlias) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing sqliteTaskAliasModel
		err := tx.Where("alias_value = ?", alias.Value).First(&existing).Error
		if err == nil {
			if existing.TaskID != alias.TaskID {
				return ErrAliasAlreadyBound
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read existing alias: %w", err)
		}

		var taskCount int64
		if err := tx.Model(&sqliteTaskModel{}).Where("task_id = ?", alias.TaskID).Count(&taskCount).Error; err != nil {
			return fmt.Errorf("verify task for alias: %w", err)
		}
		if taskCount == 0 {
			return ErrTaskNotFound
		}

		model := toAliasModel(alias)
		if err := tx.Save(&model).Error; err != nil {
			return fmt.Errorf("upsert alias: %w", err)
		}
		return nil
	})
}

func (s *SQLiteStore) ListTaskAliasRows(ctx context.Context) ([]TaskAliasRow, error) {
	models := make([]sqliteTaskAliasModel, 0)
	if err := s.db.WithContext(ctx).
		Order("updated_at DESC").
		Order("alias_value ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query task alias rows: %w", err)
	}

	result := make([]TaskAliasRow, 0, len(models))
	for _, model := range models {
		result = append(result, fromAliasModelToRow(model))
	}
	return result, nil
}

func (s *SQLiteStore) ListTaskAliasGroupCounts(ctx context.Context, groupBy string) ([]GroupCount, error) {
	column, err := taskAliasGroupByColumn(groupBy)
	if err != nil {
		return nil, err
	}

	type groupResult struct {
		Key   string `gorm:"column:key"`
		Count int    `gorm:"column:count"`
	}

	raw := make([]groupResult, 0)
	selectExpr := fmt.Sprintf("COALESCE(%s, '') as key, COUNT(1) as count", column)
	if err := s.db.WithContext(ctx).
		Model(&sqliteTaskAliasModel{}).
		Select(selectExpr).
		Group(column).
		Order("count DESC").
		Order("key ASC").
		Scan(&raw).Error; err != nil {
		return nil, fmt.Errorf("query task alias group counts: %w", err)
	}

	result := make([]GroupCount, 0, len(raw))
	for _, row := range raw {
		result = append(result, GroupCount{Key: row.Key, Count: row.Count})
	}
	return result, nil
}

func taskAliasGroupByColumn(groupBy string) (string, error) {
	switch groupBy {
	case "repo":
		return "repo", nil
	case "alias_type":
		return "alias_type", nil
	default:
		return "", fmt.Errorf("unsupported group-by %q (supported: repo, alias_type)", groupBy)
	}
}

func fromTaskModel(model sqliteTaskModel) Task {
	createdAt, _ := parseTime(model.CreatedAt)
	updatedAt, _ := parseTime(model.UpdatedAt)
	return Task{
		ID:        model.TaskID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func toAliasModel(alias TaskAlias) sqliteTaskAliasModel {
	var prNumber *int
	if alias.PRNumber > 0 {
		value := alias.PRNumber
		prNumber = &value
	}
	return sqliteTaskAliasModel{
		AliasValue: alias.Value,
		TaskID:     alias.TaskID,
		AliasType:  string(alias.Type),
		Repo:       alias.Repo,
		Branch:     alias.Branch,
		PRNumber:   prNumber,
		CreatedAt:  formatTime(alias.CreatedAt),
		UpdatedAt:  formatTime(alias.UpdatedAt),
	}
}

func fromAliasModel(model sqliteTaskAliasModel) TaskAlias {
	createdAt, _ := parseTime(model.CreatedAt)
	updatedAt, _ := parseTime(model.UpdatedAt)
	result := TaskAlias{
		TaskID:    model.TaskID,
		Type:      AliasType(model.AliasType),
		Value:     model.AliasValue,
		Repo:      model.Repo,
		Branch:    model.Branch,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if model.PRNumber != nil {
		result.PRNumber = *model.PRNumber
	}
	return result
}

func fromAliasModelToRow(model sqliteTaskAliasModel) TaskAliasRow {
	createdAt, _ := parseTime(model.CreatedAt)
	updatedAt, _ := parseTime(model.UpdatedAt)
	row := TaskAliasRow{
		TaskID:     model.TaskID,
		AliasType:  AliasType(model.AliasType),
		AliasValue: model.AliasValue,
		Repo:       model.Repo,
		Branch:     model.Branch,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
	if model.PRNumber != nil {
		row.PRNumber = *model.PRNumber
	}
	return row
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
