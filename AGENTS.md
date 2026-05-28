# AGENTS.md

## Purpose

This file contains stable project guardrails for agents working in this repo.
Keep it concise. Detailed requirements, implementation plans, and acceptance
criteria belong under `docs/specs/` and `docs/acceptance/`.

## Product Guardrails

- Build a small web scheduler for running automation jobs.
- Target Synology Container Manager / Docker.
- Preserve the single-container deployment model.
- Treat the app as single-user unless the user changes that requirement.
- Support daily and weekly schedules first.
- Initial job runtime is Bash. Leave room for future runtimes through explicit
  runtime metadata.
- Jobs may use reusable runtime-specific library recipes selected per job.
- Separate global environment from job-specific environment.
- Never store real token values in committed files, examples, logs, or docs.

Out of scope unless explicitly requested:

- Multi-user auth, roles, teams, or tenant isolation.
- Monthly schedules or interval schedules.
- External queues or distributed execution.
- Databases, unless file-based storage no longer fits.

## Persistence

All persistent state must live under a configurable mounted data directory.

Recommended environment variables:

- `APP_ADDR`: HTTP bind address. Default: `:8080`.
- `APP_DATA_DIR`: persistent data directory. Default: `/data`.
- `APP_CONFIG_PATH`: config file path. Default: `/data/config.yaml`.
- `APP_LOG_DIR`: job log directory. Default: `/data/logs`.
- `APP_SCRIPT_DIR`: job script directory. Default: `/data/scripts/jobs`.
- `APP_RECIPE_DIR`: library recipe directory. Default: `/data/recipes`.
- `APP_TIMEZONE`: scheduler timezone. Default: `Asia/Seoul`.

Recommended mounted layout:

```text
/data
  config.yaml
  logs/
  scripts/jobs/
  recipes/
```

Environment variables should provide locations, defaults, and secret values.
User-editable schedules, runtime metadata, and non-secret config belong in the
mounted config file.

## Execution And Safety

- Use structured job config rather than ad hoc command strings.
- Pass only an explicit environment allowlist to jobs.
- Secret values must be inherited by name from the container environment or a
  future secret provider.
- Capture stdout, stderr, start time, end time, exit code, and final status.
- Run the final container as a non-root user.
- Do not use `--privileged`, host networking, or Docker socket mounts.
- Do not mount broad host paths such as `/`, `/Users`, `/home`, or `/volume1`.
- Prefer a read-only container root filesystem with writable mounts only where
  the app needs them.

Arbitrary Bash jobs cannot provide strong network egress isolation by app-level
URL validation alone. Do not claim stronger sandboxing until it is enforced by a
controlled helper, proxy, or container-level policy.

## Implementation Order

Build logic before UI:

1. Config model, validation, and atomic save.
2. Daily/weekly next-run calculation.
3. Global/job environment merge and secret-name validation.
4. Runtime and recipe resolution.
5. Job runner, timeouts, log files, and run index.
6. Scheduler integration.
7. HTTP API.
8. Web UI.
9. Docker and Synology deployment docs.

Prefer focused unit tests for each logic layer before wiring it into the UI.

## Engineering Rules

- Keep changes scoped to the current implementation step.
- Use Go unless a later spec records a different decision.
- Prefer the standard library where it keeps the code clear.
- Add tests around schedule calculation, config validation, env merge, recipe
  resolution, runner behavior, and log indexing.
- Use atomic writes for config and index updates.
- When UI work starts, keep it compact and operational rather than marketing
  oriented.

