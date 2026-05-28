#!/usr/bin/env bash
set -euo pipefail

echo "Triggering ${WORKFLOW_LABEL:-GitHub Actions workflow}..."

TOKEN="${GITHUB_PAT:?GITHUB_PAT is required}"
API_BASE="https://api.github.com/repos/${OWNER:?OWNER is required}/${REPO:?REPO is required}/actions/workflows"

status_code="$(curl -sS -o /dev/null -w "%{http_code}" \
  --connect-timeout 10 \
  --max-time 30 \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "${API_BASE}/${WORKFLOW_FILE:?WORKFLOW_FILE is required}/dispatches" \
  -d "{\"ref\":\"${BRANCH:?BRANCH is required}\"}")"

echo "GitHub API status: ${status_code}"

if [[ "${status_code}" != "204" ]]; then
  echo "Workflow dispatch failed with status ${status_code}" >&2
  exit 1
fi
