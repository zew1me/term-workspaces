#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

cd_repo_root

if ! has_go_module || ! has_any_go_files; then
  echo "[golangci-lint] Skipping: no Go module/files."
  exit 0
fi

require_cmd golangci-lint

mapfile -t pkgs < <(changed_go_packages)
if [[ "${#pkgs[@]}" -eq 0 ]]; then
  echo "[golangci-lint] No changed Go packages detected; running full lint."
  golangci-lint run ./...
  exit 0
fi

golangci-lint run "${pkgs[@]}"
