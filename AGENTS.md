# AGENTS.md

## Project Intent

This project is a small single-container web scheduler for running a base bash
script on a daily or weekly schedule.

Primary target environment:

- Synology Container Manager / Docker.
- Single container deployment.
- Single user. Do not design authentication, roles, teams, or tenant isolation
  unless explicitly requested later.
- Persistent configuration and logs through bind-mounted or volume-mounted
  paths.

The product should provide a web UI that lets the operator configure schedules,
see the next run time, trigger or inspect jobs, and review execution logs.

## Product Scope

### Supported schedules

Support only these schedule types for the first version:

- Daily: run at a configured time every day.
- Weekly: run at a configured time on selected weekdays.

Weekly schedules must support selecting multiple weekdays, matching the UI
concept shown in the reference images:

- Sunday
- Monday
- Tuesday
- Wednesday
- Thursday
- Friday
- Saturday

Each schedule has exactly one run time, expressed as hour and minute.

Out of scope for now:

- Monthly schedules.
- Interval schedules such as every N minutes or every N hours.
- Multiple runs per day for a single schedule.
- Multi-user permissions.
- Distributed execution.
- External queue systems.

### Execution model

Keep execution simple:

1. The operator provides one base bash script.
2. The app stores schedule entries.
3. When a schedule fires, the app invokes the base script with schedule-specific
   context passed through environment variables or arguments.

Prefer this model over generating many separate scripts. The base script becomes
the stable integration point, while schedules remain compact structured config.

The execution contract should be explicit. A first draft:

- `JOB_ID`
- `JOB_NAME`
- `JOB_SCHEDULE_TYPE` (`daily` or `weekly`)
- `JOB_RUN_AT`
- `JOB_CONFIG_PATH`
- `JOB_LOG_DIR`

The app must capture stdout, stderr, start time, end time, exit code, and final
status for each run.

### Web UI

The web UI should support:

- Viewing all configured schedules.
- Creating, editing, enabling, and disabling a schedule.
- Daily schedule setup with time selection.
- Weekly schedule setup with weekday multi-select and time selection.
- Showing the next run time for each schedule.
- Showing recent run history and logs.
- Showing whether a job is currently running.

The UI should be operational and compact, closer to a Synology-style admin tool
than a marketing page.

## Configuration And Persistence

All persistent state must live under a configurable data directory.

Recommended environment variables:

- `APP_ADDR`: HTTP bind address. Default: `:8080`.
- `APP_DATA_DIR`: persistent data directory. Default: `/data`.
- `APP_CONFIG_PATH`: config file path. Default: `/data/config.yaml`.
- `APP_LOG_DIR`: job log directory. Default: `/data/logs`.
- `APP_SCRIPT_PATH`: base bash script path. Default: `/data/scripts/job.sh`.
- `APP_TIMEZONE`: scheduler timezone. Default: `Asia/Seoul`.

Recommended Docker volume layout:

```text
/data
  config.yaml
  logs/
  scripts/
    job.sh
```

Do not store runtime configuration only in environment variables. Environment
variables point to locations and defaults; the actual user-editable schedules
belong in the mounted config file so they survive container replacement.

## Proposed Config Shape

Use a human-editable structured config file. YAML is convenient for Synology
users editing mounted files manually, but JSON is also acceptable if the
implementation strongly prefers it.

Example:

```yaml
version: 1
timezone: Asia/Seoul
jobs:
  - id: weekday-report
    name: Weekday report
    enabled: true
    schedule:
      type: weekly
      weekdays: [monday, tuesday, wednesday, thursday, friday]
      time: "18:10"
  - id: daily-sync
    name: Daily sync
    enabled: true
    schedule:
      type: daily
      time: "02:30"
```

Implementation should validate config before saving:

- IDs are unique and stable.
- `enabled` is explicit.
- Time uses `HH:MM` 24-hour format.
- Weekly schedules have at least one weekday.
- Daily schedules do not carry weekday fields.

Use atomic writes when saving the config file.

## Logs

Persist logs under `APP_LOG_DIR`.

Suggested layout:

```text
logs/
  runs/
    2026-05-28/
      weekday-report-181000.log
  index.jsonl
```

The index should contain one JSON object per run:

- run ID
- job ID
- job name
- scheduled fire time
- actual start time
- end time
- duration
- exit code
- status
- log file path

Keep enough metadata to show recent history without scanning every log file.

## Scheduling Requirements

The scheduler must:

- Use `APP_TIMEZONE`.
- Compute and expose the next run time for each enabled job.
- Avoid duplicate execution for the same scheduled fire time.
- Not run disabled jobs.
- Serialize runs for the same job by default.
- Clearly mark skipped runs if a previous run is still active.

On process start:

- Load config from `APP_CONFIG_PATH`.
- Validate schedules.
- Register enabled jobs.
- Compute next run times.

When config changes through the web UI:

- Validate the new config.
- Save it atomically.
- Rebuild the in-memory scheduler from the saved config.

## Implementation Direction

Go is the current recommended default unless we discover a strong reason to use
another stack.

Reasons Go fits this project:

- Single static binary is easy to package in a small Docker image.
- `net/http` is enough for the backend.
- Process execution, signal handling, and log streaming are straightforward.
- Synology deployment is simpler with one binary plus mounted `/data`.

Keep dependencies conservative. Prefer the standard library where it keeps the
code clear. A cron/scheduler library is acceptable if it improves correctness,
especially around weekly schedules and timezone handling.

Frontend options, in preferred order:

1. Server-rendered HTML with small progressive JavaScript for schedule controls.
2. HTMX or a similarly small enhancement layer if it materially simplifies UI.
3. A full SPA only if the UI grows beyond the current scope.

Avoid adding a database for the first version. A config file plus log index is
enough for the stated requirements.

## Docker Direction

The final project should provide:

- `Dockerfile`
- `docker-compose.yml` or `compose.yaml` example
- README deployment notes for Synology

Expected runtime pattern:

```yaml
services:
  cron-jobs:
    image: cron-jobs:latest
    ports:
      - "8080:8080"
    environment:
      APP_ADDR: ":8080"
      APP_DATA_DIR: "/data"
      APP_CONFIG_PATH: "/data/config.yaml"
      APP_LOG_DIR: "/data/logs"
      APP_SCRIPT_PATH: "/data/scripts/job.sh"
      APP_TIMEZONE: "Asia/Seoul"
    volumes:
      - ./data:/data
```

The container must fail clearly if the base script path does not exist or is not
executable, unless the app supports creating a template script through the UI.

## Engineering Rules For Agents

- Keep the app single-user unless the user changes the requirement.
- Do not introduce external services such as Redis, Postgres, or message queues
  without explicit approval.
- Preserve the single-container deployment model.
- Keep persistent data under the configured mounted data directory.
- Use atomic config writes and avoid corrupting user-edited config.
- Treat the base bash script as the execution boundary.
- Add tests around schedule calculation, config validation, and run logging.
- When changing scheduler behavior, include examples for daily and weekly jobs.
- When adding UI, verify text does not overflow at narrow widths.

## Open Decisions

These are intentionally not finalized yet:

- Whether to use Go only with the standard library or add a scheduler package.
- YAML vs JSON for the config file.
- Whether logs should be plain files plus JSONL index, SQLite, or another local
  file format. Start with files unless requirements expand.
- Whether the web UI should allow editing the base bash script, or only show the
  configured script path.
- Whether manual "run now" should be included in the first version.
- Whether missed runs while the container is stopped should be skipped or run
  once on startup. Default assumption: skip missed runs and show the next future
  run.

