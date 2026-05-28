# Cron Jobs

A small single-container web scheduler for running daily or weekly automation
jobs. The first runtime is Bash, with reusable per-job library recipes.

## Current Capabilities

- Daily schedules at a configured `HH:MM`.
- Weekly schedules at a configured `HH:MM` on selected weekdays.
- Bash job scripts stored under the mounted data directory.
- Runtime recipes selected per job.
- Global environment and job-specific environment.
- Secret values inherited from container environment variables by name.
- Run logs and `logs/index.jsonl`.
- JSON HTTP API and a compact server-rendered web UI.

## Local Run

```sh
go test ./...
go run ./cmd/cron-jobs
```

The app listens on `:8080` by default and creates `/data/config.json` if the
configured file does not exist. For local development, set `APP_DATA_DIR` to a
writable project-local path.

```sh
APP_DATA_DIR="$PWD/data" \
APP_CONFIG_PATH="$PWD/data/config.json" \
APP_LOG_DIR="$PWD/data/logs" \
APP_SCRIPT_DIR="$PWD/data/scripts/jobs" \
APP_RECIPE_DIR="$PWD/data/recipes" \
go run ./cmd/cron-jobs
```

## Data Layout

```text
data/
  config.json
  logs/
  scripts/
    jobs/
  recipes/
```

The config stores non-secret values and inherited environment variable names.
Do not put token values in `config.json`, scripts, logs, or docs.

## GitHub Actions Dispatch Example

Prepare data files:

```sh
mkdir -p data/scripts/jobs data/recipes/bash
cp examples/config.json data/config.json
cp examples/scripts/jobs/github-workflow-dispatch.sh data/scripts/jobs/
cp examples/recipes/bash/github-actions.sh data/recipes/bash/
```

Provide the token as an environment variable:

```sh
export GITHUB_PAT="..."
```

If a token was ever pasted into chat, docs, config, or logs, revoke it in GitHub
and create a new one.

## Docker Compose

```sh
GITHUB_PAT="..." docker compose up --build
```

The compose example uses:

- non-privileged container
- default bridge networking
- no Docker socket mount
- no broad host path mount
- read-only root filesystem
- writable `/data` bind mount
- tmpfs `/tmp`

## Synology Notes

In Synology Container Manager:

1. Build or import the image.
2. Mount a host folder to `/data`.
3. Set the environment variables from `compose.yaml`.
4. Add secret values such as `GITHUB_PAT` as environment variables.
5. Publish container port `8080`.

The image runs as UID `10001`. Ensure the mounted Synology folder is writable by
that UID, or adjust ownership/permissions on the mounted directory.

## HTTP API

- `GET /api/health`
- `GET /api/config`
- `PUT /api/config`
- `GET /api/jobs`
- `POST /api/jobs/{id}/run`
- `GET /api/runs?limit=50`
- `GET /api/runs/{id}/log`

