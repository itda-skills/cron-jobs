# 0003 Implementation Plan

Status: Draft

## Strategy

Implement the core logic in small unit-tested layers before building the UI.
The UI should be a thin surface over already-tested config, scheduling, runtime,
runner, and log behavior.

## Phases

1. Project skeleton
   - Go module
   - package layout
   - minimal command entrypoint

2. Config package
   - load config
   - validate config
   - atomic save
   - tests for invalid schedules, paths, env names, recipe references

3. Schedule package
   - daily next-run calculation
   - weekly next-run calculation
   - timezone behavior
   - tests around boundary times and selected weekdays

4. Env package
   - merge global and job env
   - inherit selected container env names
   - reject invalid env names
   - tests for override and missing secret behavior

5. Runtime and recipe package
   - resolve script path
   - resolve selected recipes
   - validate language matches
   - generate Bash wrapper command

6. Runner package
   - execute job with timeout
   - constrained environment
   - controlled working directory
   - capture stdout/stderr and exit code

7. Log store package
   - write run log
   - append run index
   - query recent runs
   - read a run log

8. Scheduler integration
   - load enabled jobs
   - compute next runs
   - avoid duplicate scheduled fire times
   - serialize runs for the same job

9. HTTP API
   - list jobs
   - save config
   - next run
   - recent runs
   - log detail

10. Web UI
    - compact operational screens
    - daily/weekly controls
    - runtime, recipe, env controls
    - log viewer

11. Docker and Synology docs
    - Dockerfile
    - compose example
    - mounted data layout
    - secret environment guidance

## UI Gate

Do not start UI implementation until these packages have focused tests:

- config
- schedule
- env
- runtime/recipe
- runner
- log store

