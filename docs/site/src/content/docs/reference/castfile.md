---
title: Castfile Reference
description: Root Castfile YAML reference.
---

# Castfile

The `castfile` is the root project definition.

See the deeper references for block-specific details:

- [Task](./task) for `tasks`
- [Jobs](./jobs) for `jobs`
- [Workspace](./workspace) for nested project discovery
- [Inventory](./inventory) for hosts and SSH defaults
- [Cast Module](./module) for reusable modules
- [Remote Task](./cast-task) for `cast.task` packages

## File shape

```yaml
id: my-project
name: My Project
tasks:
  build:
    uses: bash
    run: npm run build
```

## Top-level keys

- `id`, `name`, `version`, `description`/`desc`
- `trusted_sources`, `imports`, `modules`, `config`, `defaults`
- `workspace`, `env`, `paths`, `dotenv`, `inventory`, `inventories`
- `tasks`, `jobs`, `meta`, `on`

## `id`

- Type: string
- Pattern: lowercase letters, numbers, and hyphens
- Use: stable project id; server mode sanitizes values when needed

```yaml
id: my-project
```

## `name`

- Type: string
- Use: display name and fallback source for generated ids

```yaml
name: My Project
```

## `description` / `desc`

- Type: string
- Use: human-readable description

```yaml
description: Main build project
```

## `trusted_sources`

- Type: list of strings
- Use: allowlist for remote `uses` values
- Note: remote task sources are checked against these patterns before download

```yaml
trusted_sources:
  - github.com/org/*
  - jsr:*
```

## `imports` / `modules`

- Type: list
- Shapes: string or object
- Experimental: `modules` is an alias for `imports`
- In-depth reference: [Cast Module](./module)

```yaml
imports:
  - github.com/acme/shared
  - from: github.com/acme/shared
    ns: shared
    tasks: [lint, test]
```

## `config`

- Type: object
- Fields: `context`, `contexts`, `substitution`
- `contexts` declares the available context names for the project so commands and shell completion can discover them without overloading dotenv scoping
- `substitution` controls command substitution during env/dotenv expansion; keep it off for untrusted files

```yaml
config:
  context: prod
  contexts: [dev, qa, prod]
  substitution: true
```

## `defaults`

- Type: object
- Fields: `shell`

```yaml
defaults:
  shell: bash
```

## `workspace`

- Type: boolean or object
- Object fields: `include`, `exclude`, `aliases`
- `include`/`exclude` use glob patterns
- `aliases` keys should be valid aliases
- In-depth reference: [Workspace](./workspace)

```yaml
workspace: true
```

```yaml
workspace:
  include:
    - services/**
  exclude:
    - node_modules/**
  aliases:
    backend: services/api
```

## `env`

- Type: map or ordered list
- List values may be `NAME=VALUE`, `NAME:VALUE`, or object form
- Variable values support interpolation and command substitution when substitution is enabled
- The `:` shorthand and `secret: true` mark an entry as secret metadata

```yaml
env:
  APP_ENV: prod
  API_URL: "https://api.${APP_ENV}.example.com"
  GITHUB_TOKEN: $(gh auth token)
```

```yaml
env:
  - name: DB_PASSWORD
    value: $(pass show prod/db)
    secret: true
  - APP_ENV=prod
  - API_URL:https://api.${APP_ENV}.example.com
```

Command substitution runs at load time, so only use it in trusted files. It is useful for pulling secrets into memory without storing them in the repository.

## `paths`

- Type: list
- Path entries can be strings or objects
- Object entries support `path`, `os`, and `append`
- OS-specific convenience keys like `windows`, `linux`, and `darwin` are also accepted
- Paths are resolved relative to the castfile directory
- Entries prepend to `PATH` unless `append: true` is set

```yaml
paths:
  - windows: C:/Users/path/to/dir
  - linux: /home/user/path/to/dir
  - path: /opt/tools
    append: true
```

```yaml
paths:
  - /opt/bin
  - os: linux
    path: /usr/local/bin
  - windows: C:/Users/path/to/dir
  - linux: /home/user/path/to/dir
  - path: /opt/tools
    append: true
```

## `dotenv`

- Type: list
- Each entry is a path or object
- Optional files: prefix or suffix with `?`
- Context-scoped files use `contexts`
- Dotenv files are expanded with the same interpolation and command substitution rules as `env`
- Use optional files to keep local-only values out of the required set

```yaml
dotenv:
  - path: .env
  - path: ?.env.local
  - path: .env.prod
    contexts: [prod]
  - os: windows
    path: .env.windows
```

```dotenv
APP_ENV=prod
API_URL=https://api.${APP_ENV}.example.com
SECRET=$(gh auth token)
```

## `inventory`

- Type: object
- Fields: `defaults`, `hosts`
- `hosts` can be a map or a sequence of host entries
- In-depth reference: [Inventory](./inventory)

```yaml
inventory:
  defaults:
    ssh:
      user: deploy
  hosts:
    api:
      host: 10.0.0.10
      defaults: ssh
```

## `inventories`

- Type: list of strings
- Use: load and merge extra inventory files

## `tasks`

- Type: map of task names to task definitions
- Naming: task keys may include `:`, `.`, `/`, `@`, `-`, and `_`
- Common `uses` values: `bash`, `sh`, `shell`, `docker`, `node`, `deno`, `bun`, `python`, `pwsh`, `powershell`, `go`, `dotnet`, `csharp`, `ssh`, `scp`, `tmpl`, `cast`
- In-depth reference: [Task](./task)
- Use scalar values for quick shell commands, or mappings for runner-specific settings

```yaml
tasks:
  build:
    uses: bash
    run: npm run build
```

## `jobs`

- Type: map of job names to job definitions
- Experimental
- In-depth reference: [Jobs](./jobs)

```yaml
jobs:
  deploy:
    steps:
      - run: deploy
```

## `meta`

- Type: free-form mapping
- Use: arbitrary project metadata

## `on`

- Type: object
- Fields: `schedule`, `webhooks`
- Use this block when running Cast as a server

```yaml
on:
  schedule:
    crons: ["0 0 * * *"]
  webhooks:
    deploy:
      job: deploy
      secret: super-secret
```

## Notes

- `workspace: false` disables workspace discovery.
- `dotenv` entries marked with `?` are optional and skipped if missing.
- Jobs, modules, and remote task definitions are experimental.
