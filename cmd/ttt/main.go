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
	case "link-pr":
		return runTaskLinkPR(args[1:])
	default:
		return printTaskUsage()
	}
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

func printUsage() error {
	fmt.Println("ttt usage:")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}

func printTaskUsage() error {
	fmt.Println("ttt task usage:")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}
