# term-workspaces

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
