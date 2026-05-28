# MVP Acceptance Plan

Status: Draft

## Config

- [ ] Loads a valid config from `APP_CONFIG_PATH`.
- [ ] Rejects duplicate job IDs.
- [ ] Rejects invalid time values.
- [ ] Rejects weekly schedules with no weekdays.
- [ ] Rejects script paths outside `APP_SCRIPT_DIR`.
- [ ] Rejects recipe paths outside `APP_RECIPE_DIR`.
- [ ] Rejects selected recipes whose language differs from the job runtime.

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
- [ ] Selected Bash recipes are sourced before the job script.
- [ ] Job timeout terminates the process.
- [ ] stdout and stderr are captured.
- [ ] exit code and status are recorded.
- [ ] Job working directory is controlled by the app.

## Logs

- [ ] Each run has a log file.
- [ ] Each run appends metadata to `logs/index.jsonl`.
- [ ] Recent runs can be listed without scanning every log file.
- [ ] A selected run log can be read by run ID.

## UI

- [ ] UI is implemented only after core unit-tested logic exists.
- [ ] UI can create and edit daily jobs.
- [ ] UI can create and edit weekly jobs with multiple weekdays.
- [ ] UI can select runtime recipes.
- [ ] UI separates global env from job env.
- [ ] UI shows next run time.
- [ ] UI shows recent run history and log detail.

