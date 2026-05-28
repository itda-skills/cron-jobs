# MVP Acceptance Plan

Status: Draft

## Config

- [ ] Loads a valid config from `APP_CONFIG_PATH`.
- [ ] Creates new web UI jobs with generated UUIDv7-style IDs.
- [ ] Rejects duplicate job IDs.
- [ ] Rejects invalid time values.
- [ ] Rejects weekly schedules with no weekdays.
- [ ] Rejects script paths outside `APP_SCRIPT_DIR`.

## Schedule

- [ ] Daily job scheduled later today returns today's run time.
- [ ] Daily job scheduled earlier today returns tomorrow's run time.
- [ ] Weekly job returns the next selected weekday at the configured time.
- [ ] Weekly job moves to the next week after the final selected weekday passes.
- [ ] Next-run calculation uses `APP_TIMEZONE`.

## Environment

- [ ] Global plain env applies to every job.
- [ ] Job plain env overrides global plain env.
- [ ] Global inherited env is available to every job when present.
- [ ] Job inherited env is available only to that job when present.
- [ ] Missing inherited env produces a clear validation or execution error.
- [ ] Secret values are not written to config examples or logs.

## Runner

- [ ] Bash job receives only the explicit environment allowlist plus `JOB_*`.
- [ ] Job timeout terminates the process.
- [ ] stdout and stderr are captured.
- [ ] exit code and status are recorded.
- [ ] Job working directory is controlled by the app.
- [ ] Test runs set `JOB_RUN_REASON=test` and `JOB_TEST_RUN=true`.

## Logs

- [ ] Each run has a log file.
- [ ] Each run appends metadata to `logs/index.jsonl`.
- [ ] Recent runs can be listed without scanning every log file.
- [ ] A selected run log can be read by run ID.

## UI

- [ ] UI is implemented only after core unit-tested logic exists.
- [ ] UI can create and edit daily jobs.
- [ ] UI can create and edit weekly jobs with multiple weekdays.
- [ ] UI can edit full job script content.
- [ ] UI can test-run draft job values before saving.
- [ ] UI shows test-run output.
- [ ] UI does not ask for app-managed job IDs or saved script paths.
- [ ] UI separates global env from job env.
- [ ] UI shows next run time.
- [ ] UI shows recent run history and log detail.
