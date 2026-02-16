#!/usr/bin/env bash
set -euo pipefail

repo_root() {
  git rev-parse --show-toplevel
}

cd_repo_root() {
  cd "$(repo_root)"
}

setup_go_cache_env() {
  local root
  root="$(repo_root)"

  export GOCACHE="${GOCACHE:-${root}/.cache/go-build}"
  export GOMODCACHE="${GOMODCACHE:-${root}/.cache/go-mod}"
  export XDG_CACHE_HOME="${XDG_CACHE_HOME:-${root}/.cache}"
  export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-${root}/.cache/golangci-lint}"

  mkdir -p "${GOCACHE}" "${GOMODCACHE}" "${XDG_CACHE_HOME}" "${GOLANGCI_LINT_CACHE}"
}

has_go_module() {
  [[ -f go.mod ]]
}

has_any_go_files() {
  git ls-files '*.go' | grep -q .
}

upstream_ref() {
  git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null || true
}

collect_changed_go_files() {
  local staged
  staged="$(git diff --cached --name-only -- '*.go')"
  if [[ -n "${staged}" ]]; then
    printf '%s\n' "${staged}"
    return 0
  fi

  local upstream
  upstream="$(upstream_ref)"
  if [[ -n "${upstream}" ]]; then
    local base
    base="$(git merge-base HEAD "${upstream}")"
    git diff --name-only "${base}"...HEAD -- '*.go'
    return 0
  fi

  # Fall back to current branch delta when no upstream is configured.
  git diff --name-only HEAD~1..HEAD -- '*.go' 2>/dev/null || true
}

changed_go_packages() {
  local files
  files="$(collect_changed_go_files)"

  if [[ -z "${files}" ]]; then
    return 0
  fi

  local dirs
  dirs="$(printf '%s\n' "${files}" | xargs -I{} dirname {} | sed 's#^#./#' | sed 's#^\./\.$#./#' | sort -u)"

  if [[ -n "${dirs}" ]]; then
    printf '%s\n' "${dirs}"
  fi
}

require_cmd() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}" >&2
    exit 1
  fi
}
