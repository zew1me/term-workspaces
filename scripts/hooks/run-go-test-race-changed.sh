#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

cd_repo_root

if ! has_go_module || ! has_any_go_files; then
  echo "[go-test-race] Skipping: no Go module/files."
  exit 0
fi

require_cmd go

mapfile -t pkgs < <(changed_go_packages)
if [[ "${#pkgs[@]}" -eq 0 ]]; then
  echo "[go-test-race] No changed Go packages detected; running full race tests."
  go test --race ./...
  exit 0
fi

go test --race "${pkgs[@]}"
