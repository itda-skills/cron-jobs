#!/usr/bin/env bash

github_actions_dispatch() {
  local owner="${OWNER:?OWNER is required}"
  local repo="${REPO:?REPO is required}"
  local workflow_file="${WORKFLOW_FILE:?WORKFLOW_FILE is required}"
  local branch="${BRANCH:?BRANCH is required}"
  local token="${GITHUB_PAT:?GITHUB_PAT is required}"
  local api_base="https://api.github.com/repos/${owner}/${repo}/actions/workflows"

  curl -sS -o /dev/null -w "%{http_code}\n" \
    --connect-timeout 10 \
    --max-time 30 \
    -X POST \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer ${token}" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "${api_base}/${workflow_file}/dispatches" \
    -d "{\"ref\":\"${branch}\"}"
}

