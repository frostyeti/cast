---
title: Task Reference
description: Task YAML reference for castfile, modules, and remote task handlers.
---

# Task

Tasks are the main execution unit in Cast. Use the mapping form for full control, or the scalar form for quick shell commands.

For remote package metadata, see [Remote Task](./cast-task).

## Shapes

```yaml
tasks:
  build: npm run build
```

```yaml
tasks:
  build:
    uses: bash
    run: npm run build
```

## Fields

### `id`

- Purpose: stable task id for lookups, server mode, and cross-project references.
- Example: `id: build-app`

```yaml
tasks:
  build:
    id: build-app
    run: npm run build
```

### `name`

- Purpose: display label and fallback source for derived ids.
- Example: `name: Build App`

### `slug`

- Purpose: short canonical label for lookups and display.
- Example: `slug: build-app`

### `desc` / `description`

- Purpose: short summary shown in task lists.
- Example: `desc: Compile the app`

### `help`

- Purpose: longer help text for humans.
- Example: `help: Builds the app, runs tests, and publishes artifacts.`

### `uses`

- Purpose: selects the runner or remote source for the task.
- Common values: `bash`, `sh`, `shell`, `pwsh`, `powershell`, `node`, `deno`, `bun`, `python`, `ruby`, `go`, `golang`, `dotnet`, `csharp`, `docker`, `ssh`, `scp`, `tmpl`, `cast`.
- Remote forms: GitHub, JSR, npm, local paths, and file URLs.

```yaml
tasks:
  build:
    uses: bash
    run: npm run build

  remote-lint:
    uses: github.com/acme/task-pack@v1.2.0

  local-script:
    uses: ./scripts/task.ts
```

### `run`

- Purpose: shell command, script body, or runner entrypoint.
- Example: `run: npm run build`

```yaml
tasks:
  build:
    uses: bash
    run: |
      set -euo pipefail
      npm install
      npm run build

  script-file:
    uses: bash
    run: ./scripts/build.sh
```

### `args`

- Purpose: positional args passed to the selected runner.
- Example: `args: ["--release", "--verbose"]`

### `env`

- Purpose: task-local environment variables.
- Example: `env: { NODE_ENV: production }`
- Supports variable interpolation and command substitution.

```yaml
tasks:
  deploy:
    env:
      APP_ENV: prod
      API_URL: "https://api.${APP_ENV}.example.com"
      SECRET_TOKEN: $(gh auth token)
    run: echo "$API_URL"
```

Command substitution is powerful and dangerous: only use it in trusted files. It lets you fetch secrets at load time without storing them in the repository.

### `dotenv`

- Purpose: dotenv files to load before the task runs.
- Optional files: prefix or suffix a path with `?`.
- Example: `dotenv: ["?.env.local", ".env.production"]`

```yaml
tasks:
  start:
    dotenv:
      - .env
      - ?.env.local
      - path: .env.prod
        contexts: [prod]
    run: node server.js
```

### `cwd`

- Purpose: working directory before execution.
- Example: `cwd: ./web`

### `timeout`

- Purpose: duration limit for the task.
- Example: `timeout: 5m`

### `needs`

- Purpose: task dependencies that must run first.
- Shapes: scalar or list; each dependency may also include `parallel: true`.

```yaml
tasks:
  deploy:
    needs: build
    run: ./scripts/deploy.sh

  test:
    needs:
      - id: build
      - id: lint
        parallel: true
    run: npm test
```

### `with`

- Purpose: runner-specific settings and inputs.
- Common keys:
  - `script` for shell and SSH tasks
  - `image`, `command`, `args`, `volumes` for Docker tasks
  - `files`, `values`, `disable-env`, `disable-gotmpl` for `tmpl`
  - `file`, `dir`, `task`, `job` for `cast`
  - `max-parallel` for `ssh` and `scp`

```yaml
tasks:
  shell-file:
    uses: bash
    with:
      script: ./scripts/build.sh

  docker-build:
    uses: docker
    with:
      image: node:20
      command: npm
      args: [test]

  render-config:
    uses: tmpl
    with:
      files:
        - ./templates/app.yml.tmpl:./out/app.yml
      values: ./values.yml

  run-frontend:
    uses: cast
    with:
      dir: ./frontend
      task: build
```

### `hosts`

- Purpose: limit execution to named inventory hosts.
- Example: `hosts: [web-1, web-2]`

### `if`

- Purpose: runtime predicate that decides whether the task runs.
- Example: `if: env.BRANCH == 'main'`

### `hooks`

- Purpose: before/after hook task names.
- Boolean form enables the default `task-id:before` and `task-id:after` hooks.

```yaml
tasks:
  build:
    hooks:
      before: pre-build
      after: post-build
    run: npm run build
```

### `force`

- Purpose: override skip behavior when `if` or dependencies would stop the task.
- Example: `force: env.ALWAYS_RUN == 'true'`

### `extends`

- Purpose: inherit settings from another task.
- Child values override the base; environment, `with`, and dotenv content are merged.

```yaml
tasks:
  base-build:
    cwd: ./app
    env:
      NODE_ENV: production
    run: npm run build

  build-dev:
    extends: base-build
    env:
      NODE_ENV: development
```

### `template`

- Purpose: render `run` with Go templates before execution.
- `true` is shorthand for `gotmpl`.

```yaml
tasks:
  deploy:
    template: gotmpl
    run: |
      echo "Deploying {{ .env.APP_ENV }}"
```

## Runner examples

### Shell scripts

```yaml
tasks:
  lint:
    uses: bash
    run: |
      set -euo pipefail
      npm run lint

  build-sh:
    uses: sh
    run: ./scripts/build.sh

  windows-script:
    uses: pwsh
    run: ./scripts/build.ps1
```

### Bun, Deno, and Node

```yaml
tasks:
  serve-bun:
    uses: bun
    run: ./scripts/serve.ts

  test-deno:
    uses: deno
    run: ./scripts/test.ts

  cli-node:
    uses: node
    run: ./scripts/cli.js
```

These runners can execute inline code too:

```yaml
tasks:
  quick-deno:
    uses: deno
    run: |
      console.log("hello from deno")

  quick-bun:
    uses: bun
    run: |
      console.log("hello from bun")
```

### Dotnet and C#

```yaml
tasks:
  build-dotnet:
    uses: dotnet
    run: ./src/App/App.csproj

  run-csharp:
    uses: csharp
    run: ./scripts/Hello.cs

  inline-cs:
    uses: dotnet
    run: |
      using System;
      Console.WriteLine("Hello from C#");
```

`dotnet` and `csharp` can run `.cs`, `.csx`, `.dll`, `.exe`, and project files such as `.csproj`, `.fsproj`, and `.vbproj`.

### Docker

```yaml
tasks:
  test-in-docker:
    uses: docker
    with:
      image: node:20
    run: npm test
```

### SSH, SCP, and templates

```yaml
tasks:
  deploy:
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

  render-files:
    uses: tmpl
    with:
      files:
        - ./templates/app.yml.tmpl:./out/app.yml
      values: ./values.yml
```

### Remote tasks and cross-project execution

```yaml
trusted_sources:
  - github.com/acme/*

tasks:
  remote-lint:
    uses: github.com/acme/task-pack@v1.2.0

  frontend-build:
    uses: cast
    with:
      dir: ./frontend
      task: build

  backend-release:
    uses: cast
    with:
      file: ./backend/castfile.yaml
      job: release
```

## Notes

- Task keys may contain `:`, `.`, `/`, `@`, `-`, and `_`.
- Keep ids lowercase and hyphenated for predictable lookup.
- Remote task sources should be allowlisted with `trusted_sources`.
- `dotenv` entries with `?` are optional and skipped when missing.

## Task help and `--help`

Cast supports task-level help output in direct and dynamic subcommand flows.

- `cast test:bun --help` prints task `help`, then falls back to `desc`.
- `cast test bun --help` behaves the same when `subcmds` is configured.
- Namespace subcommands can expose a `help` task (for example `mysql:help`) for `cast mysql --help`.

```yaml
tasks:
  test:bun:
    help: |
      Runs bun tests with the project defaults.
    desc: Run bun tests
    uses: shell
    run: bun test
```

## Context suffix resolution

Task lookup is context-aware.

- `cast -c prod deploy` tries `deploy:prod`, then `deploy`.
- `cast run -c prod deploy` follows the same rule.

Use this pattern to keep one logical task name while splitting environment behavior by suffix.

## Hook resolution and order

Hooks resolve by suffix against the task id:

- `before: [before]` resolves to `<task-id>:before`
- `after: [after]` resolves to `<task-id>:after`

Execution order is:

1. dependencies (`needs`)
2. before hooks
3. main task
4. after hooks

This allows consistent setup/teardown around both base and context-suffixed tasks.

## `with` as task inputs

`with` is the parameter map for task handlers and remote tasks.

- Remote `cast.task` inputs map to `INPUT_*` environment variables.
- `docker` reads image/command/args/volumes from `with`.
- `ssh` and `scp` use `with` keys like `script`, `files`, and `max-parallel`.
- `cast` (cross-project) uses `with.file`, `with.dir`, `with.task`, `with.job`.

Keep keys stable so callers can pass predictable values.

## Argument passthrough

Trailing CLI args are passed into task execution. For shell and docker style tasks, trailing args are appended to the effective command args in supported paths.

```bash
cast test:bun -- --clean ./tmp
cast test bun --clean ./tmp
cast run test:bun --clean ./tmp
```

## `uses: shell` details

`uses: shell` is the minimal built-in command runner.

- no language-specific wrapper required
- supports simple command execution and script-like multi-line blocks
- supports command operators in script mode (`&&`, `||`, `|`, `;`)
- supports `with.script` for loading script content from files
- supports `template: gotmpl` to render `run` before execution

```yaml
tasks:
  verify:
    uses: shell
    template: gotmpl
    run: |
      echo "context={{ .env.CAST_CONTEXT }}"
      npm run lint && npm run test
```

## Remote `uses` deep dive

Supported remote styles include:

- `gh:` / `github:`
- `gl:` / `gitlab:`
- `azdo:`
- `cast:` / `task:` / `spell:` (spells namespace)
- hosted forms like `github.com/org/repo@...`
- direct SSH/URL forms such as `git@host:org/repo.git@ref/subpath` and `ssh://...@ref`

Version and ref behavior:

- exact tags (`@v1.2.3`)
- semver family resolution (`@v1`, `@v1.2`)
- prerelease exact (`@v2.3.1-beta.1`)
- branch refs (`@main`, `@feature/x`)
- commit SHA refs (`@abc1234`)
- `@head` / `@HEAD`

Subpaths are supported after the ref (`@v1.2.3/tasks/lint`). Cast validates subpaths and uses sparse checkout for subpath-targeted fetches.

Use `trusted_sources` in your castfile to allowlist remote refs.

## Environment, dotenv, and paths cascade

Environment is assembled in layers and merged over time:

1. process env
2. imported/project `paths`
3. imported/project `dotenv` (context-aware)
4. imported/project `env`
5. task `dotenv`
6. task `env`

Top-level `paths` modify PATH for all tasks, and Cast prepends `./bin` and `./node_modules/.bin`.

## Runtime chaining with `CAST_ENV` and `CAST_PATH`

Tasks can update later tasks by writing to runtime files:

- write `KEY=value` lines to `$CAST_ENV`
- write directory lines to `$CAST_PATH`

Values written by one successful task are loaded into the shared project env/PATH for following tasks. This is useful for dynamic secrets and late overrides.

```yaml
tasks:
  load-secrets:
    uses: shell
    run: echo "API_TOKEN=$(gh auth token)" >> "$CAST_ENV"

  add-tools:
    uses: shell
    run: echo "./tools/bin" >> "$CAST_PATH"
```

## SSH and SCP fan-out

`ssh` and `scp` tasks can target multiple hosts by explicit host name or by matching host tags in `hosts`.

- tasks fan out across targets using parallel workers
- control concurrency with `with.max-parallel`
- `ssh` supports `template: gotmpl` for script rendering

This model is designed for asynchronous multi-host operations with bounded parallelism.
