#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

cd_repo_root

if ! has_go_module || ! has_any_go_files; then
  echo "[go-format] Skipping: no Go module/files."
  exit 0
fi

require_cmd go
require_cmd gofumpt

go fmt ./...
gofumpt -w .

changed="$(git diff --name-only -- '*.go')"
if [[ -n "${changed}" ]]; then
  echo "[go-format] Formatting changed files. Re-stage and commit again:" >&2
  printf '%s\n' "${changed}" >&2
  exit 1
fi
