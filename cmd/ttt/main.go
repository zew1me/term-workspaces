package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"term-workspaces/internal/tasks"
	"term-workspaces/internal/wezterm"
	"time"
)

var newWezTermClient = func() wezterm.Client {
	return wezterm.NewCLIClient()
}

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
	case "dashboard":
		return runTaskDashboard(args[1:])
	case "close-session":
		return runTaskCloseSession(args[1:])
	case "ensure-prepr":
		return runTaskEnsurePrePR(args[1:])
	case "ensure-note":
		return runTaskEnsureNote(args[1:])
	case "list":
		return runTaskList(args[1:])
	case "open-session":
		return runTaskOpenSession(args[1:])
	case "open-note":
		return runTaskOpenNote(args[1:])
	case "sessions":
		return runTaskSessions(args[1:])
	case "link-pr":
		return runTaskLinkPR(args[1:])
	default:
		return printTaskUsage()
	}
}

type dashboardPayload struct {
	Groups       dashboardGroups            `json:"groups"`
	Sessions     []tasks.TaskSession        `json:"sessions"`
	OpenSessions []tasks.TaskSession        `json:"open_sessions"`
	Aliases      []tasks.TaskAliasRow       `json:"aliases"`
	Tasks        []dashboardTaskMergedEntry `json:"tasks"`
}

type dashboardGroups struct {
	ByRepo          []tasks.GroupCount `json:"by_repo"`
	ByAliasType     []tasks.GroupCount `json:"by_alias_type"`
	BySessionStatus []tasks.GroupCount `json:"by_session_status"`
}

type dashboardTaskMergedEntry struct {
	Task    tasks.Task           `json:"task"`
	Aliases []tasks.TaskAliasRow `json:"aliases"`
	Session *tasks.TaskSession   `json:"session,omitempty"`
}

func runTaskDashboard(args []string) error {
	fs := flag.NewFlagSet("task dashboard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	jsonOutput := fs.Bool("json", true, "Emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	ctx := context.Background()
	byRepo, err := store.ListTaskAliasGroupCounts(ctx, "repo")
	if err != nil {
		return fmt.Errorf("dashboard repo groups: %w", err)
	}
	byAliasType, err := store.ListTaskAliasGroupCounts(ctx, "alias_type")
	if err != nil {
		return fmt.Errorf("dashboard alias_type groups: %w", err)
	}
	bySessionStatus, err := store.ListSessionStatusCounts(ctx)
	if err != nil {
		return fmt.Errorf("dashboard session status groups: %w", err)
	}
	aliases, err := store.ListTaskAliasRows(ctx)
	if err != nil {
		return fmt.Errorf("dashboard aliases: %w", err)
	}
	taskRows, err := store.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("dashboard tasks: %w", err)
	}
	sessions, err := store.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("dashboard sessions: %w", err)
	}

	payload := dashboardPayload{
		Groups: dashboardGroups{
			ByRepo:          byRepo,
			ByAliasType:     byAliasType,
			BySessionStatus: bySessionStatus,
		},
		Sessions:     sessions,
		OpenSessions: filterOpenSessions(sessions),
		Aliases:      aliases,
		Tasks:        mergeDashboardTaskRows(taskRows, aliases, sessions),
	}

	if *jsonOutput {
		return writeJSON(payload)
	}

	fmt.Printf("repos=%d alias_types=%d session_statuses=%d aliases=%d sessions=%d open_sessions=%d tasks=%d\n",
		len(byRepo), len(byAliasType), len(bySessionStatus), len(aliases), len(sessions), len(payload.OpenSessions), len(payload.Tasks))
	return nil
}

func runTaskSessions(args []string) error {
	fs := flag.NewFlagSet("task sessions", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON")
	groupBy := fs.String("group-by", "", "Group sessions by metadata: status")
	reconcile := fs.Bool("reconcile", false, "Reconcile session health against live WezTerm panes before output")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()
	ctx := context.Background()
	if *reconcile {
		if err := reconcileSessionHealth(ctx, store, newWezTermClient()); err != nil {
			return fmt.Errorf("reconcile sessions: %w", err)
		}
	}

	if *groupBy != "" {
		if *groupBy != "status" {
			return fmt.Errorf("unsupported group-by %q (supported: status)", *groupBy)
		}
		groups, err := store.ListSessionStatusCounts(ctx)
		if err != nil {
			return fmt.Errorf("list session groups: %w", err)
		}
		if len(groups) == 0 {
			fmt.Println("no sessions")
			return nil
		}
		if *jsonOutput {
			return writeJSON(groups)
		}
		fmt.Println("group_key\tcount")
		for _, entry := range groups {
			fmt.Printf("%s\t%d\n", entry.Key, entry.Count)
		}
		return nil
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("no sessions")
		return nil
	}
	if *jsonOutput {
		return writeJSON(sessions)
	}

	fmt.Println("task_id\tstatus\tworkspace\tpane_id\tcwd\tcommand\tcodex_session_id")
	for _, session := range sessions {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			session.TaskID,
			session.Status,
			session.Workspace,
			session.PaneID,
			session.Cwd,
			session.Command,
			session.CodexSessionID,
		)
	}
	return nil
}

func runTaskOpenSession(args []string) error {
	fs := flag.NewFlagSet("task open-session", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Branch name (optional when using --pr)")
	prNumber := fs.Int("pr", 0, "Pull request number (optional when using --branch)")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	cwd := fs.String("cwd", ".", "Working directory for spawned session")
	workspace := fs.String("workspace", "", "Override workspace name")
	command := fs.String("command", "codex", "Session command metadata label")

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
	task, err := resolveTaskForNote(context.Background(), service, *repo, *branch, *prNumber)
	if err != nil {
		return err
	}

	client := newWezTermClient()
	ctx := context.Background()
	now := time.Now().UTC()
	existing, found, err := store.GetSessionByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("load existing session: %w", err)
	}

	if found && existing.PaneID > 0 {
		panes, err := client.ListPanes(ctx)
		if err != nil {
			return fmt.Errorf("list panes for liveness check: %w", err)
		}
		if !paneIDPresent(panes, existing.PaneID) {
			existing.Status = tasks.SessionStatusClosed
			existing.PaneID = 0
			existing.UpdatedAt = now
			if err := store.UpsertSession(ctx, existing); err != nil {
				return fmt.Errorf("persist stale session: %w", err)
			}
		} else if err := client.ActivatePane(ctx, existing.PaneID); err == nil {
			existing.Status = tasks.SessionStatusOpen
			existing.LastSeenAt = now
			existing.UpdatedAt = now
			if err := store.UpsertSession(ctx, existing); err != nil {
				return fmt.Errorf("persist activated session: %w", err)
			}
			fmt.Printf("task_id=%s status=activated pane_id=%d workspace=%s\n", task.ID, existing.PaneID, existing.Workspace)
			return nil
		}
	}

	targetWorkspace := strings.TrimSpace(*workspace)
	if targetWorkspace == "" {
		if found && strings.TrimSpace(existing.Workspace) != "" {
			targetWorkspace = existing.Workspace
		} else {
			targetWorkspace = workspaceForTaskID(task.ID)
		}
	}

	paneID, err := client.Spawn(ctx, targetWorkspace, *cwd)
	if err != nil {
		return fmt.Errorf("spawn session pane: %w", err)
	}

	session := tasks.TaskSession{
		TaskID:         task.ID,
		Workspace:      targetWorkspace,
		PaneID:         paneID,
		Cwd:            *cwd,
		Command:        *command,
		Status:         tasks.SessionStatusOpen,
		CodexSessionID: "",
		LastSeenAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if found {
		session.CreatedAt = existing.CreatedAt
		if session.CreatedAt.IsZero() {
			session.CreatedAt = now
		}
	}
	if err := store.UpsertSession(ctx, session); err != nil {
		return fmt.Errorf("persist spawned session: %w", err)
	}

	fmt.Printf("task_id=%s status=spawned pane_id=%d workspace=%s\n", task.ID, paneID, targetWorkspace)
	return nil
}

func runTaskCloseSession(args []string) error {
	fs := flag.NewFlagSet("task close-session", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Branch name (optional when using --pr)")
	prNumber := fs.Int("pr", 0, "Pull request number (optional when using --branch)")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
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
	task, err := resolveTaskForNote(context.Background(), service, *repo, *branch, *prNumber)
	if err != nil {
		return err
	}

	ctx := context.Background()
	session, found, err := store.GetSessionByTaskID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("load existing session: %w", err)
	}
	if !found {
		fmt.Printf("task_id=%s status=missing\n", task.ID)
		return nil
	}

	client := newWezTermClient()
	if session.PaneID > 0 {
		if err := client.KillPane(ctx, session.PaneID); err != nil {
			return fmt.Errorf("kill pane %d: %w", session.PaneID, err)
		}
	}

	now := time.Now().UTC()
	session.Status = tasks.SessionStatusClosed
	// Policy: retain workspace/cwd/command metadata but clear stale pane binding.
	session.PaneID = 0
	session.UpdatedAt = now
	if err := store.UpsertSession(ctx, session); err != nil {
		return fmt.Errorf("persist closed session: %w", err)
	}

	fmt.Printf("task_id=%s status=closed workspace=%s\n", task.ID, session.Workspace)
	return nil
}

func runTaskList(args []string) error {
	fs := flag.NewFlagSet("task list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	groupBy := fs.String("group-by", "", "Group task aliases by metadata: repo, alias_type")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := tasks.NewSQLiteStore(*dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite task store: %w", err)
	}
	defer func() {
		_ = store.Close()
	}()

	if *groupBy != "" {
		groups, err := store.ListTaskAliasGroupCounts(context.Background(), *groupBy)
		if err != nil {
			return fmt.Errorf("list task groups: %w", err)
		}
		if len(groups) == 0 {
			fmt.Println("no tasks")
			return nil
		}
		if *jsonOutput {
			return writeJSON(groups)
		}

		fmt.Println("group_key\tcount")
		for _, entry := range groups {
			fmt.Printf("%s\t%d\n", entry.Key, entry.Count)
		}
		return nil
	}

	rows, err := store.ListTaskAliasRows(context.Background())
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}
	if len(rows) == 0 {
		fmt.Println("no tasks")
		return nil
	}
	if *jsonOutput {
		return writeJSON(rows)
	}

	fmt.Println("task_id\talias_type\talias_value\trepo\tbranch\tpr")
	for _, row := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%d\n",
			row.TaskID,
			row.AliasType,
			row.AliasValue,
			row.Repo,
			row.Branch,
			row.PRNumber,
		)
	}
	return nil
}

func writeJSON(value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal json output: %w", err)
	}
	fmt.Println(string(encoded))
	return nil
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
	task, err := resolveTaskForNote(context.Background(), service, *repo, *branch, *prNumber)
	if err != nil {
		return err
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

func runTaskOpenNote(args []string) error {
	fs := flag.NewFlagSet("task open-note", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	repo := fs.String("repo", "", "GitHub repository in owner/repo format")
	branch := fs.String("branch", "", "Branch name (optional when using --pr)")
	prNumber := fs.Int("pr", 0, "Pull request number (optional when using --branch)")
	dbPath := fs.String("db", defaultDBPath(), "Path to sqlite database")
	notesDir := fs.String("notes-dir", defaultNotesDir(), "Directory for task note markdown files")
	dryRun := fs.Bool("dry-run", false, "Print editor command without launching")

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
	task, err := resolveTaskForNote(context.Background(), service, *repo, *branch, *prNumber)
	if err != nil {
		return err
	}

	path, _, err := tasks.EnsureTaskNote(*notesDir, task.ID)
	if err != nil {
		return fmt.Errorf("ensure task note: %w", err)
	}

	editorName, editorArgs := tasks.ResolveEditorCommand(os.Getenv("EDITOR"), path)
	if *dryRun {
		fmt.Printf("task_id=%s status=dry_run note_path=%s editor=%s args=%v\n", task.ID, path, editorName, editorArgs)
		return nil
	}

	// #nosec G204 -- editor command is intentionally user-configurable via $EDITOR.
	command := exec.Command(editorName, editorArgs...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("open note with editor: %w", err)
	}

	fmt.Printf("task_id=%s status=opened note_path=%s editor=%s\n", task.ID, path, editorName)
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

func workspaceForTaskID(taskID string) string {
	sanitized := strings.NewReplacer("/", "-", ":", "-", "#", "-").Replace(taskID)
	return "task-" + sanitized
}

func filterOpenSessions(sessions []tasks.TaskSession) []tasks.TaskSession {
	filtered := make([]tasks.TaskSession, 0, len(sessions))
	for _, session := range sessions {
		if session.Status == tasks.SessionStatusOpen {
			filtered = append(filtered, session)
		}
	}
	return filtered
}

func mergeDashboardTaskRows(
	taskRows []tasks.Task,
	aliases []tasks.TaskAliasRow,
	sessions []tasks.TaskSession,
) []dashboardTaskMergedEntry {
	aliasesByTask := make(map[string][]tasks.TaskAliasRow, len(taskRows))
	for _, alias := range aliases {
		aliasesByTask[alias.TaskID] = append(aliasesByTask[alias.TaskID], alias)
	}

	sessionsByTask := make(map[string]tasks.TaskSession, len(sessions))
	for _, session := range sessions {
		sessionsByTask[session.TaskID] = session
	}

	result := make([]dashboardTaskMergedEntry, 0, len(taskRows))
	for _, row := range taskRows {
		entry := dashboardTaskMergedEntry{
			Task:    row,
			Aliases: aliasesByTask[row.ID],
		}
		if session, ok := sessionsByTask[row.ID]; ok {
			sessionCopy := session
			entry.Session = &sessionCopy
		}
		result = append(result, entry)
	}
	return result
}

func paneIDPresent(panes []wezterm.Pane, paneID int64) bool {
	for _, pane := range panes {
		if pane.PaneID == paneID {
			return true
		}
	}
	return false
}

func reconcileSessionHealth(ctx context.Context, store *tasks.SQLiteStore, client wezterm.Client) error {
	panes, err := client.ListPanes(ctx)
	if err != nil {
		return err
	}
	sessions, err := store.ListSessions(ctx)
	if err != nil {
		return err
	}
	paneSet := make(map[int64]struct{}, len(panes))
	for _, pane := range panes {
		paneSet[pane.PaneID] = struct{}{}
	}

	now := time.Now().UTC()
	for _, session := range sessions {
		original := session.Status
		next := original
		switch {
		case session.PaneID <= 0:
			if session.Status != tasks.SessionStatusClosed {
				next = tasks.SessionStatusUnknown
			}
		case paneIDPresentMap(paneSet, session.PaneID):
			next = tasks.SessionStatusOpen
			session.LastSeenAt = now
		default:
			next = tasks.SessionStatusClosed
		}
		if next == original {
			continue
		}
		session.Status = next
		session.UpdatedAt = now
		if err := store.UpsertSession(ctx, session); err != nil {
			return err
		}
	}
	return nil
}

func paneIDPresentMap(panes map[int64]struct{}, paneID int64) bool {
	_, ok := panes[paneID]
	return ok
}

func resolveTaskForNote(ctx context.Context, service *tasks.Service, repo, branch string, prNumber int) (tasks.Task, error) {
	switch {
	case branch != "" && prNumber > 0:
		linkedTask, _, linkErr := service.LinkPRToPrePR(ctx, repo, branch, prNumber)
		if linkErr != nil {
			return tasks.Task{}, fmt.Errorf("link pr to task: %w", linkErr)
		}
		return linkedTask, nil
	case branch != "":
		preTask, _, preErr := service.GetOrCreatePrePRTask(ctx, repo, branch)
		if preErr != nil {
			return tasks.Task{}, fmt.Errorf("ensure pre-pr task: %w", preErr)
		}
		return preTask, nil
	default:
		prTask, found, prErr := service.GetTaskByPR(ctx, repo, prNumber)
		if prErr != nil {
			return tasks.Task{}, fmt.Errorf("resolve pr task: %w", prErr)
		}
		if !found {
			return tasks.Task{}, fmt.Errorf("no task found for %s; link it first with `ttt task link-pr`", tasks.PRAliasValue(repo, prNumber))
		}
		return prTask, nil
	}
}

func printUsage() error {
	fmt.Println("ttt usage:")
	fmt.Println("  ttt task ensure-prepr --repo owner/repo --branch feature/name [--db path]")
	fmt.Println("  ttt task close-session --repo owner/repo [--branch feature/name] [--pr 123] [--db path]")
	fmt.Println("  ttt task dashboard [--db path] [--json]")
	fmt.Println("  ttt task ensure-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path]")
	fmt.Println("  ttt task list [--db path] [--group-by repo|alias_type] [--json]")
	fmt.Println("  ttt task open-session --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--cwd path] [--workspace name] [--command label]")
	fmt.Println("  ttt task open-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path] [--dry-run]")
	fmt.Println("  ttt task sessions [--db path] [--group-by status] [--reconcile] [--json]")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}

func printTaskUsage() error {
	fmt.Println("ttt task usage:")
	fmt.Println("  ttt task ensure-prepr --repo owner/repo --branch feature/name [--db path]")
	fmt.Println("  ttt task close-session --repo owner/repo [--branch feature/name] [--pr 123] [--db path]")
	fmt.Println("  ttt task dashboard [--db path] [--json]")
	fmt.Println("  ttt task ensure-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path]")
	fmt.Println("  ttt task list [--db path] [--group-by repo|alias_type] [--json]")
	fmt.Println("  ttt task open-session --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--cwd path] [--workspace name] [--command label]")
	fmt.Println("  ttt task open-note --repo owner/repo [--branch feature/name] [--pr 123] [--db path] [--notes-dir path] [--dry-run]")
	fmt.Println("  ttt task sessions [--db path] [--group-by status] [--reconcile] [--json]")
	fmt.Println("  ttt task link-pr --repo owner/repo --branch feature/name --pr 123 [--db path]")
	return nil
}
