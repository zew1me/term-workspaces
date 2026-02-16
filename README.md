# term-workspaces

## Task CLI
`ttt` currently supports task identity linking, note scaffolding, and task listing backed by SQLite.

### Common Commands
```bash
# create/find a pre-PR task identity for repo+branch
go run ./cmd/ttt task ensure-prepr --repo owner/repo --branch feature/name

# link a PR alias to an existing pre-PR task (or create from PR)
go run ./cmd/ttt task link-pr --repo owner/repo --branch feature/name --pr 123

# create/find the task note markdown file
go run ./cmd/ttt task ensure-note --repo owner/repo --branch feature/name

# open task note in $EDITOR (or open -e), dry-run supported
go run ./cmd/ttt task open-note --repo owner/repo --branch feature/name --dry-run

# list task aliases
go run ./cmd/ttt task list

# grouped list views for top-level summaries
go run ./cmd/ttt task list --group-by repo
go run ./cmd/ttt task list --group-by alias_type
```

## Go Hook Tooling
This repo uses `prek` for local Git hooks.

### Setup
```bash
./scripts/setup-hooks.sh
```

### Manual Runs
```bash
prek list
prek run --all-files --stage pre-commit
prek run --all-files --stage pre-push
```

### Tooling Included
- formatting: `go fmt`, `gofumpt -w .`
- linting: `golangci-lint` (includes `govet`, `staticcheck`, `gosec`, `gofumpt` checks)
- tests: `go test`, `go test --race`
- build: `go build`
- vuln scan: `govulncheck`

Hooks auto-skip when no `go.mod` or no Go files exist.
