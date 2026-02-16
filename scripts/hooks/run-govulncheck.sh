#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

cd_repo_root
setup_go_cache_env

if ! has_go_module || ! has_any_go_files; then
  echo "[govulncheck] Skipping: no Go module/files."
  exit 0
fi

require_cmd govulncheck

govulncheck ./...
