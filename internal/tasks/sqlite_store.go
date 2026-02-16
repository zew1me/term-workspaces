package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite parent dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	store, err := NewSQLiteStoreFromDB(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func NewSQLiteStoreFromDB(db *sql.DB) (*SQLiteStore, error) {
	store := &SQLiteStore{db: db}
	if err := store.migrate(context.Background()); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
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
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate sqlite schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) CreateTask(ctx context.Context, task Task) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tasks(task_id, created_at, updated_at) VALUES (?, ?, ?)`,
		task.ID,
		formatTime(task.CreatedAt),
		formatTime(task.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetTask(ctx context.Context, taskID string) (Task, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT task_id, created_at, updated_at FROM tasks WHERE task_id = ?`,
		taskID,
	)

	return scanTask(row)
}

func (s *SQLiteStore) GetTaskByAlias(ctx context.Context, aliasValue string) (Task, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT t.task_id, t.created_at, t.updated_at
		 FROM tasks t
		 JOIN task_aliases a ON a.task_id = t.task_id
		 WHERE a.alias_value = ?`,
		aliasValue,
	)

	return scanTask(row)
}

func (s *SQLiteStore) GetAlias(ctx context.Context, aliasValue string) (TaskAlias, bool, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT task_id, alias_type, alias_value, repo, branch, pr_number, created_at, updated_at
		 FROM task_aliases
		 WHERE alias_value = ?`,
		aliasValue,
	)

	var (
		alias          TaskAlias
		aliasType      string
		repo, branch   sql.NullString
		prNumber       sql.NullInt64
		createdAtRaw   string
		updatedAtRaw   string
	)

	err := row.Scan(
		&alias.TaskID,
		&aliasType,
		&alias.Value,
		&repo,
		&branch,
		&prNumber,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskAlias{}, false, nil
	}
	if err != nil {
		return TaskAlias{}, false, fmt.Errorf("query alias: %w", err)
	}

	alias.Type = AliasType(aliasType)
	alias.Repo = repo.String
	alias.Branch = branch.String
	if prNumber.Valid {
		alias.PRNumber = int(prNumber.Int64)
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return TaskAlias{}, false, fmt.Errorf("parse alias created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return TaskAlias{}, false, fmt.Errorf("parse alias updated_at: %w", err)
	}
	alias.CreatedAt = createdAt
	alias.UpdatedAt = updatedAt

	return alias, true, nil
}

func (s *SQLiteStore) UpsertAlias(ctx context.Context, alias TaskAlias) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start alias upsert tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingTaskID string
	row := tx.QueryRowContext(
		ctx,
		`SELECT task_id FROM task_aliases WHERE alias_value = ?`,
		alias.Value,
	)
	switch scanErr := row.Scan(&existingTaskID); {
	case errors.Is(scanErr, sql.ErrNoRows):
	case scanErr != nil:
		return fmt.Errorf("read existing alias: %w", scanErr)
	default:
		if existingTaskID != alias.TaskID {
			return ErrAliasAlreadyBound
		}
	}

	var taskExists int
	taskRow := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM tasks WHERE task_id = ?`, alias.TaskID)
	if err := taskRow.Scan(&taskExists); err != nil {
		return fmt.Errorf("verify task for alias: %w", err)
	}
	if taskExists == 0 {
		return ErrTaskNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO task_aliases(alias_value, task_id, alias_type, repo, branch, pr_number, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(alias_value) DO UPDATE SET
		   task_id = excluded.task_id,
		   alias_type = excluded.alias_type,
		   repo = excluded.repo,
		   branch = excluded.branch,
		   pr_number = excluded.pr_number,
		   updated_at = excluded.updated_at`,
		alias.Value,
		alias.TaskID,
		string(alias.Type),
		nullIfEmpty(alias.Repo),
		nullIfEmpty(alias.Branch),
		nullIfZero(alias.PRNumber),
		formatTime(alias.CreatedAt),
		formatTime(alias.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert alias: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit alias upsert tx: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ListTaskAliasRows(ctx context.Context) ([]TaskAliasRow, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT a.task_id, a.alias_type, a.alias_value, a.repo, a.branch, a.pr_number, a.created_at, a.updated_at
		 FROM task_aliases a
		 ORDER BY a.updated_at DESC, a.alias_value ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query task alias rows: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]TaskAliasRow, 0)
	for rows.Next() {
		var (
			row                TaskAliasRow
			aliasType          string
			repo, branch       sql.NullString
			prNumber           sql.NullInt64
			createdAtRaw       string
			updatedAtRaw       string
		)

		if err := rows.Scan(
			&row.TaskID,
			&aliasType,
			&row.AliasValue,
			&repo,
			&branch,
			&prNumber,
			&createdAtRaw,
			&updatedAtRaw,
		); err != nil {
			return nil, fmt.Errorf("scan task alias row: %w", err)
		}

		row.AliasType = AliasType(aliasType)
		row.Repo = repo.String
		row.Branch = branch.String
		if prNumber.Valid {
			row.PRNumber = int(prNumber.Int64)
		}

		createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse task alias created_at: %w", err)
		}
		updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse task alias updated_at: %w", err)
		}
		row.CreatedAt = createdAt
		row.UpdatedAt = updatedAt

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task alias rows: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) ListTaskAliasGroupCounts(ctx context.Context, groupBy string) ([]GroupCount, error) {
	column, err := taskAliasGroupByColumn(groupBy)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		`SELECT COALESCE(%s, ''), COUNT(1)
		 FROM task_aliases
		 GROUP BY 1
		 ORDER BY COUNT(1) DESC, 1 ASC`,
		column,
	)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query task alias group counts: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]GroupCount, 0)
	for rows.Next() {
		var entry GroupCount
		if err := rows.Scan(&entry.Key, &entry.Count); err != nil {
			return nil, fmt.Errorf("scan task alias group count: %w", err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task alias group counts: %w", err)
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

func scanTask(row *sql.Row) (Task, bool, error) {
	var (
		task      Task
		createdAt string
		updatedAt string
	)

	err := row.Scan(&task.ID, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, fmt.Errorf("scan task: %w", err)
	}

	parsedCreated, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Task{}, false, fmt.Errorf("parse task created_at: %w", err)
	}
	parsedUpdated, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Task{}, false, fmt.Errorf("parse task updated_at: %w", err)
	}

	task.CreatedAt = parsedCreated
	task.UpdatedAt = parsedUpdated

	return task, true, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullIfZero(value int) any {
	if value == 0 {
		return nil
	}
	return value
}
