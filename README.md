# Cast

Cast is a task runner and automation tool for local scripts, remote execution, and reusable remote task packages.

## Installation

### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.ps1 | iex
```

By default, Cast installs to `~/.local/bin` on Linux/macOS and `~/AppData/Local/Programs/bin` on Windows.

## Shell completion

The completion command is available and hidden from root help. You can still run:

```bash
cast completion bash
cast completion zsh
cast completion fish
cast completion powershell
```

Example (bash):

```bash
source <(cast completion bash)
echo 'source <(cast completion bash)' >> ~/.bashrc
```

## Quick start

Create a `castfile`:

```yaml
name: demo
tasks:
  hello:
    uses: shell
    run: echo "hello"
```

Run:

```bash
cast list
cast hello
cast run hello
```

## Task execution model

### Context suffix routing

Cast resolves a context-specific task first, then falls back to the base task.

- `cast -c prod deploy` tries `deploy:prod`, then `deploy`
- `cast run deploy -c prod` behaves the same

### Hooks (`before` and `after`)

Hooks are resolved as `task-id:<hook-suffix>`. For example:

```yaml
tasks:
  deploy:
    hooks:
      before: [before]
      after: [after]
    uses: shell
    run: echo deploy

  deploy:before:
    uses: shell
    run: echo pre

  deploy:after:
    uses: shell
    run: echo post
```

Execution order is: dependencies -> before hooks -> main task -> after hooks.

### `with` inputs

`with` is the input/parameter map for handlers and remote tasks.

- `cast` tasks use `with.file`, `with.dir`, `with.task`, `with.job`
- `docker` tasks use `with.image`, `with.command`, `with.args`, `with.volumes`
- `ssh`/`scp` often use `with.script`, `with.files`, `with.max-parallel`
- remote `cast.task` inputs map into `INPUT_*` environment variables

## `uses: shell` in detail

`uses: shell` is the most lightweight runner. It does not require language-specific wrappers and can run basic commands directly.

- supports single commands and script-style multi-line run blocks
- supports operators in script mode (`&&`, `||`, `|`, `;`)
- supports `template: gotmpl` for rendering `run` before execution
- can read script content via `with.script`

Example:

```yaml
tasks:
  build:
    uses: shell
    template: gotmpl
    run: |
      echo "building {{ .env.APP_ENV }}"
      npm run build && npm run test
```

### CLI arg passthrough

Trailing CLI args are passed to tasks as task args.

- `cast test:bun -- --clean ./tmp`
- dynamic subcommands also pass args, for example `cast test bun --clean ./tmp`

For simple `shell`/`docker` invocations, those trailing args are appended to command args.

## Help behavior (`--help`)

For task execution paths, `--help` can show task-level docs.

- direct task: `cast test:bun --help`
- dynamic subcommand leaf: `cast test bun --help`
- subcommand root help task pattern: `cast mysql --help` can use `mysql:help`

Behavior:

1. print task `help` when present
2. otherwise print task `desc`
3. otherwise print task id/name fallback

## Subcommands (`subcmds`)

You can expose task namespaces as CLI subcommands:

```yaml
subcmds:
  - test
  - mysql

tasks:
  test:bun:
    help: run bun tests
    uses: shell
    run: bun test
```

Usage:

- `cast test bun`
- `cast test bun --help`
- `cast mysql --help` (uses `mysql:help` if defined)

Alias key `subcommands` is also accepted.

## Remote tasks (`uses:` remote refs)

Cast supports remote task sources via `uses`. Common prefixes:

- `gh:` or `github:` for GitHub repos
- `gl:` or `gitlab:` for GitLab repos
- `azdo:` for Azure DevOps repos
- `cast:`, `task:`, `spell:` for the default spells repo namespace
- direct hosts: `github.com/org/repo@...`, `gitlab.com/group/repo@...`, `dev.azure.com/...`
- URLs and SSH clone refs: `https://...git@ref`, `ssh://...`, `git@host:org/repo.git@ref`

### Trust model

Set `trusted_sources` to allow remote `uses` patterns. If non-empty, remote refs must match.

```yaml
trusted_sources:
  - gh:your-org/*
  - cast:*
  - github.com/frostyeti/*
```

### Versioning and refs

Remote refs support:

- exact tags: `@v1.2.3`
- semver family resolution: `@v1` (resolves to best matching tag)
- prerelease exact matches: `@v2.3.1-beta.1`
- branch names: `@main`, `@master`, `@feature/x`
- commit SHAs: `@abc1234` (7-40 hex)
- `@head` / `@HEAD`

### Stable vs volatile cache

- immutable refs (exact tags/commits) use stable cache
- branch/head-like refs use volatile cache

Use:

- `cast task install` to prefetch remote tasks
- `cast task update` to refresh branch/head refs
- `cast task clear-cache` to clear local volatile cache
- `cast task clear-cache --global` to clear global stable cache

### Subpaths

Remote refs can include a subpath after the version, for example:

```yaml
tasks:
  lint:
    uses: gh:org/automation@v1.2.3/tasks/lint
```

Cast performs sparse checkout for subpath-targeted refs and prevents traversal segments like `..`.

### SSH clone examples

```yaml
tasks:
  private-task:
    uses: git@github.com:your-org/private-tasks.git@v1.2.0/path/to/task

  private-task-ssh-url:
    uses: ssh://git@github.com/your-org/private-tasks.git@main/tasks/build
```

## Environment and dotenv cascading

Cast builds task environment in layers:

1. process environment
2. imported module `paths`, project `paths`
3. imported module `dotenv`, project `dotenv` (context-aware)
4. imported module `env`, project `env`
5. task `dotenv`
6. task `env`

Later layers override earlier values.

### `paths` cascade

Top-level `paths` entries are applied to `PATH` for task execution (prepend by default, optional append).

Cast also prepends `./bin` and `./node_modules/.bin` for convenience.

### Runtime env propagation with `CAST_ENV` and `CAST_PATH`

Cast creates/uses files for cross-task propagation:

- `CAST_ENV`: write `KEY=value` lines to inject env vars into subsequent tasks
- `CAST_PATH`: write path lines to prepend directories to PATH for subsequent tasks
- `CAST_OUTPUTS`: write outputs for task result sharing

Example:

```yaml
tasks:
  bootstrap-secrets:
    uses: shell
    run: |
      echo "API_TOKEN=$(gh auth token)" >> "$CAST_ENV"

  setup-tools:
    uses: shell
    run: |
      echo "./tools/bin" >> "$CAST_PATH"

  use-both:
    needs: [bootstrap-secrets, setup-tools]
    uses: shell
    run: |
      echo "$API_TOKEN"
      my-tool --version
```

This pattern is useful for dynamic secret loading and late env overrides without committing secret values.

## Templating support

- `shell` tasks support `template: gotmpl`
- `ssh` tasks support `template: gotmpl`

You can interpolate environment values and host data where supported.

## SSH and SCP task targeting and concurrency

`ssh` and `scp` tasks can target multiple hosts by explicit host ids or by tag selectors in `hosts`.

- multi-target operations run concurrently via worker pools
- default concurrency is bounded (override with `with.max-parallel` or env vars)

Example:

```yaml
tasks:
  rollout:
    uses: ssh
    hosts: [web, api, prod] # names or tags
    with:
      max-parallel: "10"
    run: ./deploy.sh

  sync-assets:
    uses: scp
    hosts: [web]
    with:
      max-parallel: "8"
      files:
        - ./dist:/srv/app/dist
```

## CLI commands you will likely use

```bash
cast list
cast <task>
cast run <task>
cast run --job <job>
cast task install
cast task update
cast task clear-cache
cast ssh <host>
cast scp <src> <dest> --targets host1,host2
```

## More docs

- `docs/site/src/content/docs/reference/castfile.md`
- `docs/site/src/content/docs/reference/task.md`
- `docs/site/src/content/docs/reference/cast.md`
- `docs/site/src/content/docs/guides/ssh-and-scp.md`
