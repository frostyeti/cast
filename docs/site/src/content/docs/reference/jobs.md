---
title: Jobs Reference
description: Experimental job YAML reference.
---

# Jobs

Jobs are experimental pipeline definitions made of ordered steps.

See [Task](./task) for the task fields referenced by scalar step names.

## Shapes

```yaml
jobs:
  deploy:
    steps:
      - build
      - render-config
      - sync-assets
      - deploy-prod
```

## Fields

### `id`

- Purpose: stable job id for lookups and server mode.
- Example: `id: deploy`

### `name`

- Purpose: display label for listings.

### `desc`

- Purpose: short summary shown in job lists.

### `needs`

- Purpose: upstream job dependencies.
- Shapes: scalar or list of dependency objects.

```yaml
jobs:
  build:
    steps:
      - build-app

  deploy:
    needs: build
    steps:
      - deploy-app
```

### `steps`

- Purpose: ordered execution plan.
- Today, scalar step names are what actually run; mapping steps are parsed, but step-level `run`/`uses` execution is still reserved.
- Use task names here to chain built-in runners and remote tasks.

```yaml
jobs:
  release:
    steps:
      - build-app
      - render-config
      - sync-assets
      - deploy-prod
      - remote-lint
```

```yaml
jobs:
  release:
    steps:
      - build-app
      - render-config
      - sync-assets
      - deploy-prod
      - remote-lint
```

### `env`

- Purpose: job-scoped environment variables.
- Supports interpolation and command substitution like task-level `env`.

### `dotenv`

- Purpose: dotenv file list for the job.
- Optional files: prefix or suffix with `?`.
- Same expansion rules as task-level `dotenv`.

### `if`

- Purpose: runtime predicate for whether the job should run.

### `timeout`

- Purpose: duration limit for the whole job.

### `cwd`

- Purpose: working directory for job steps.

### `extends`

- Purpose: inherit from another job.
- Child values override the base; `steps`, `env`, and `dotenv` are merged.

```yaml
jobs:
  base-deploy:
    cwd: ./deploy
    timeout: 10m
    steps:
      - render-config

  deploy-prod:
    extends: base-deploy
    env:
      REGION: us-east-1
```

### `cron`

- Purpose: legacy single cron expression for the job.
- Prefer `on.schedule.crons` for project-level schedules.

## Built-in task examples used by jobs

Jobs usually reference tasks that are already configured with built-in runners.

```yaml
tasks:
  build-app:
    uses: docker
    with:
      image: node:20
    run: npm run build

  deploy-prod:
    uses: ssh
    hosts: [prod]
    run: |
      cd /srv/app
      docker compose up -d

  sync-assets:
    uses: scp
    hosts: [prod]
    with:
      files:
        - ./dist:/srv/app/dist

  render-config:
    uses: tmpl
    with:
      files:
        - ./templates/app.yml.tmpl:./out/app.yml
      values: ./values.yml

  remote-lint:
    uses: github.com/acme/task-pack@v1.2.0

jobs:
  release:
    steps:
      - build-app
      - render-config
      - sync-assets
      - deploy-prod
      - remote-lint
```

## Notes

- Jobs are experimental.
- Job ids should be lowercase and hyphenated.
- Scalar step names inherit task lookup rules and point at tasks.
- Use `needs` for job ordering; use `steps` for in-job sequencing.
