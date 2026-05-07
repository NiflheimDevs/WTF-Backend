#!/usr/bin/env sh
set -eu

remote_name="${1:-remote}"
repo_root=$(git rev-parse --show-toplevel)

printf '%s\n' "pre-push: running make test before pushing to ${remote_name}..."
(
  cd "$repo_root"
  make test
)
