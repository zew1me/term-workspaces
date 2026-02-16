package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"term-workspaces/internal/tasks"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ttt error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "task":
		return runTask(args[1:])
	default:
		return printUsage()
	}
}

func runTask(args []string) error {
	if len(args) == 0 {
		return printTaskUsage()
	}

	switch args[0] {
	case "ensure-prepr":
		return runTaskEnsurePrePR(args[1:])
	case "ensure-note":
		return runTaskEnsureNote(args[1:])
	case "link-pr":
		return runTaskLinkPR(args[1:])
	default:
		return printTaskUsage()
	}
}

func runTaskEnsurePrePR(args []string) error {
	fs := flag.NewFlagSet("task ensure-prepr", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Pre-PR branch name")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *branch == "" {
		return fmt.Errorf("--repo and --branch are required")
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	service := tasks.NewService(store)
	task, created, err := service.GetOrCreatePrePRTask(context.Background(), *repo, *branch)
	if err != nil {
		return fmt.Errorf("ensure pre-pr task: %w", err)
	}

	status := "existing"
	if created {
		status = "created"
	}
	fmt.Printf("task_id=%s status=%s prepr_alias=%s\n",
		task.ID,
		status,
		tasks.PrePRAliasValue(*repo, *branch),
	)
	return nil
}

func runTaskEnsureNote(args []string) error {
	fs := flag.NewFlagSet("task ensure-note", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Branch name (optional when using --pr)")
	prNumber := fs.Int("pr", 0, "Pull request number (optional when using --branch)")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	notesDir := fs.String("notes-dir", defaultNotesDir(), "Directory for task note markdown files")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" {
		return fmt.Errorf("--repo is required")
	}
	if *branch == "" && *prNumber <= 0 {
		return fmt.Errorf("one of --branch or --pr is required")
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	service := tasks.NewService(store)
	ctx := context.Background()

	var task tasks.Task
	switch {
	case *branch != "" && *prNumber > 0:
		linkedTask, _, linkErr := service.LinkPRToPrePR(ctx, *repo, *branch, *prNumber)
		if linkErr != nil {
			return fmt.Errorf("link pr to task: %w", linkErr)
		}
		task = linkedTask
	case *branch != "":
		preTask, _, preErr := service.GetOrCreatePrePRTask(ctx, *repo, *branch)
		if preErr != nil {
			return fmt.Errorf("ensure pre-pr task: %w", preErr)
		}
		task = preTask
	default:
		prTask, found, prErr := service.GetTaskByPR(ctx, *repo, *prNumber)
		if prErr != nil {
			return fmt.Errorf("resolve pr task: %w", prErr)
		}
		if !found {
			return fmt.Errorf("no task found for %s; link it first with `ttt task link-pr`", tasks.PRAliasValue(*repo, *prNumber))
		}
		task = prTask
	}

	path, created, err := tasks.EnsureTaskNote(*notesDir, task.ID)
	if err != nil {
		return fmt.Errorf("ensure task note: %w", err)
	}

	status := "existing"
	if created {
		status = "created"
	}
	fmt.Printf("task_id=%s status=%s note_path=%s\n", task.ID, status, path)
	return nil
}

func runTaskLinkPR(args []string) error {
	fs := flag.NewFlagSet("task link-pr", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Pre-PR branch name")
	prNumber := fs.Int("pr", 0, "Pull request number")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *repo == "" || *branch == "" || *prNumber <= 0 {
		return fmt.Errorf("--repo, --branch, and --pr are required")
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	service := tasks.NewService(store)
	task, status, err := service.LinkPRToPrePR(context.Background(), *repo, *branch, *prNumber)
	if err != nil {
		return fmt.Errorf("link pr to task: %w", err)
	}

	fmt.Printf("task_id=%s status=%s pr_alias=%s prepr_alias=%s\n",
		task.ID,
		status,
		tasks.PRAliasValue(*repo, *prNumber),
		tasks.PrePRAliasValue(*repo, *branch),
	)
	return nil
}

func defaultDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ttt/state.db"
	}
	return filepath.Join(home, "Library", "Application Support", "ttt", "state.db")
}

func defaultNotesDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ttt/notes"
	}
	return filepath.Join(home, "Library", "Application Support", "ttt", "notes")
}

func printUsage() error {
	fmt.Println("ttt usage:")
	fmt.Println("  ttt task ensure-prepr --repo owner/repo --branch feature/name [--db path]")
	fmt.Println("  ttt task ensure-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path]")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}

func printTaskUsage() error {
	fmt.Println("ttt task usage:")
	fmt.Println("  ttt task ensure-prepr --repo owner/repo --branch feature/name [--db path]")
	fmt.Println("  ttt task ensure-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path]")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}
