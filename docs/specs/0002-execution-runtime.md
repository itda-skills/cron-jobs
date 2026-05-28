# 0002 Execution Runtime

Status: Draft

## Goal

Support scheduled execution of user-provided Bash scripts in a single-container
Synology deployment, while leaving room for future runtimes beyond Bash.

## Runtime Model

Initial runtime:

- `language: bash`
- app-managed script stored under `APP_SCRIPT_DIR`
- timeout required

Runtime config shape:

```json
{
  "runtime": {
    "language": "bash",
    "script": "scripts/jobs/weekday-report.sh",
    "timeout_seconds": 60
  }
}
```

The web UI should accept full script content. On save, the app generates the job
ID when needed, writes the content to an app-managed script path under
`APP_SCRIPT_DIR`, and saves metadata in `config.json`.

## Test Run Model

New and edited jobs should support a test run before saving.

Rules:

- Test run uses the current form values and script content.
- Test run does not save the job config.
- Test run writes a temporary script under `APP_SCRIPT_DIR`.
- Test run captures stdout, stderr, status, and exit code.
- Test run output is shown directly in the UI.
- Test run entries may be written to the run log index, marked as test runs.

The runner should set:

- `JOB_RUN_REASON=test` for test runs.
- `JOB_TEST_RUN=true` for test runs.

Scheduled or manual saved-job runs should set:

- `JOB_RUN_REASON=scheduled` or `manual`.
- `JOB_TEST_RUN=false`.

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

## Bash Job Contract

The runner should provide these `JOB_*` variables:

- `JOB_ID`
- `JOB_NAME`
- `JOB_SCHEDULE_TYPE`
- `JOB_RUN_AT`
- `JOB_CONFIG_PATH`
- `JOB_LOG_DIR`
- `JOB_SCRIPT_PATH`
- `JOB_RUN_REASON`
- `JOB_TEST_RUN`

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
