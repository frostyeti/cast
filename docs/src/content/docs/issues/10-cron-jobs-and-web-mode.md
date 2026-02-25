---
title: "Cron Jobs and Web Mode"
type: feature
tags: ["feature", "cron", "web"]
---

# Cron Jobs and Web Server Mode

## Overview

Cast should support automated scheduling of tasks via Cron expressions and offer a long-running web server mode (`cast web`). This allows Cast to function as a lightweight, declarative cron daemon and remote task execution engine.

## 1. Schema Updates

Add support for a top-level `jobs` block and a schedule trigger in the `castfile` schema. Jobs orchestrate the execution of one or more tasks and manage their own dependencies (`needs`), environments, and conditional logic (`if`).

### Example Syntax:

```yaml
on:
  schedule:
    crons: ["0 0 * * *", "*/15 * * * *"] # Execute daily at midnight, and every 15 mins

tasks:
  one: 
    uses: bash
    run: echo "task one"
  next-task:
    uses: bash
    run: echo "task two"
  four:
    uses: bash
    run: echo "task four"

jobs:
  default:
    name: "Nightly Build" # Overrides display name
    steps: ["one", "next-task"]
    env:
      BUILD_ENV: "production"
    dotenv: [".env.prod"]
    if: "true" # Predicate similar to tasks

  next_job:
    needs: ["default"]
    steps: ["four"]
```

## 2. Web Mode (`cast web`)

Introduce a new CLI command to start the Cast server:

```bash
cast web --port 8080 --addr 127.0.0.1
```

### File Discovery
When the web server starts, it should parse and load Castfiles from the following locations:
- The `castfile` in the Current Working Directory (CWD).
- `/etc/cast/jobs/*.yaml` (Linux/macOS)
- `C:\ProgramData\cast\jobs\*.yaml` (Windows)
- `./cast/jobs/*.yaml` or `./.cast/jobs/*.yaml` relative to CWD.

*Note: All discovered files must conform to the standard `castfile` schema.*

### Project ID/Name Generation
If a discovered `castfile` does not have an explicit `id` or `name` field:
- Use the filename (if not named `castfile.yaml` or `castfile.yml`).
- If named `castfile.yaml`, use the `basename` of its parent directory.
- Replace any spaces in the resulting string with hyphens `-`.

### Execution Isolation
Similar to standard task execution, when a job runs on a schedule or via API trigger, its environment variables (`env` / `dotenv`) must remain strictly isolated. They **must not** leak into the parent `cast web` daemon process or affect other concurrently running jobs.

*Implementation Note: Use a reliable scheduler module like `github.com/go-co-op/gocron/v2` to manage the cron evaluations and trigger execution pipelines.*

## 3. HTTP API Endpoints

The web server should expose a simple REST API to interact with the loaded projects, jobs, and tasks.

**General:**
- `GET /health` : Returns 200 OK (basic health check).

**Projects:**
- `GET /api/v1/projects` : Returns a list of all loaded Castfiles with their generated/explicit `id` and `name`.
- `GET /api/v1/projects/{id}/jobs` : Returns a list of jobs defined in the specified project.
- `GET /api/v1/projects/{id}/tasks` : Returns a list of tasks defined in the specified project.

**Triggers:**
- `POST /api/v1/projects/{id}/jobs/{jobId}/trigger` : Triggers a specific job to run on-the-fly.
- `POST /api/v1/projects/{id}/tasks/{taskId}/trigger` : Triggers a specific task to run on-the-fly.
