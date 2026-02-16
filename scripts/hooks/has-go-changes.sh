#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "${SCRIPT_DIR}/common.sh"

cd_repo_root

if ! has_go_module || ! has_any_go_files; then
  exit 1
fi

if changed_go_packages | grep -q .; then
  exit 0
fi

exit 1
