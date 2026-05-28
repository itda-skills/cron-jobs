#!/usr/bin/env bash
set -euo pipefail

echo "Triggering ${WORKFLOW_LABEL:-GitHub Actions workflow}..."
status_code="$(github_actions_dispatch)"
echo "GitHub API status: ${status_code}"

if [[ "${status_code}" != "204" ]]; then
  echo "Workflow dispatch failed with status ${status_code}" >&2
  exit 1
fi

