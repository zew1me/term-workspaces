#!/usr/bin/env bash
set -euo pipefail

if ! command -v prek >/dev/null 2>&1; then
  echo "Error: 'prek' is not installed. Install it first: https://prek.j178.dev" >&2
  exit 1
fi

# Allow override for sandboxed environments.
: "${PREK_HOME:=${HOME}/.cache/prek}"
mkdir -p "${PREK_HOME}"

prek install
prek install-hooks

echo "Installed prek hooks for this repository."
