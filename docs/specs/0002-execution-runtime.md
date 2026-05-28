# 0002 Execution Runtime

Status: Draft

## Goal

Support scheduled script execution safely enough for a single-container Synology
deployment, while leaving room for future runtimes beyond Bash.

## Runtime Model

Initial runtime:

- `language: bash`
- script stored under `APP_SCRIPT_DIR`
- selected recipes loaded before the job script
- timeout required

Future runtimes should be represented with the same shape:

```json
{
  "runtime": {
    "language": "bash",
    "script": "scripts/jobs/weekday-report.sh",
    "recipes": ["github-actions"],
    "timeout_seconds": 60
  }
}
```

## Environment Model

Separate global and job-specific environment.

```json
{
  "env": {
    "global": {
      "plain": {
        "BRANCH": "main"
      },
      "inherit": ["GITHUB_PAT"]
    }
  }
}
```

```json
{
  "jobs": [
    {
      "id": "weekday-report",
      "env": {
        "plain": {
          "OWNER": "itda-skills",
          "REPO": "rs-golden-queens",
          "WORKFLOW_FILE": "flow-kr.yml"
        },
        "inherit": []
      }
    }
  ]
}
```

Rules:

- Plain values are non-secret values stored in config.
- Inherited values are variable names loaded from the container environment.
- Job plain values override global plain values.
- Job inherited names add to global inherited names.
- Missing inherited variables should fail validation or prevent execution with a
  clear error.
- Real token values must not be stored in config, scripts, docs, or logs.

## Recipe Model

Recipes are reusable runtime-specific helpers.

```json
{
  "recipes": [
    {
      "id": "github-actions",
      "language": "bash",
      "path": "recipes/bash/github-actions.sh"
    }
  ]
}
```

Rules:

- Jobs select recipes by ID.
- Selected recipe language must match the job runtime language.
- Bash recipes are sourced before the job script in listed order.
- Bash recipes should define functions and constants only; they should not run
  work at source time.

Initial built-in recipe candidates:

- `http-curl`
- `github-actions`

## Bash Job Contract

The runner should provide these `JOB_*` variables:

- `JOB_ID`
- `JOB_NAME`
- `JOB_SCHEDULE_TYPE`
- `JOB_RUN_AT`
- `JOB_CONFIG_PATH`
- `JOB_LOG_DIR`
- `JOB_SCRIPT_PATH`

Example script shape:

```bash
#!/usr/bin/env bash
set -euo pipefail

TOKEN="${GITHUB_PAT:?GITHUB_PAT is required}"
API_BASE="https://api.github.com/repos/${OWNER}/${REPO}/actions/workflows"

curl -sS -o /dev/null -w "%{http_code}\n" \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "${API_BASE}/${WORKFLOW_FILE}/dispatches" \
  -d "{\"ref\":\"${BRANCH}\"}"
```

## Safety Notes

The container should be non-root, non-privileged, and avoid broad host mounts.

Arbitrary Bash scripts can execute any command available in the container and
can read mounted app data. Strong network egress controls require a controlled
HTTP helper, proxy, or container-level policy.
