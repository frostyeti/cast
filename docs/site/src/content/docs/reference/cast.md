---
title: Remote Task Reference
description: Experimental cast YAML reference.
---

# Remote Task

`cast` or `cast.yaml` defines a remote task package.

## Shapes

```yaml
name: format
description: Format source files
inputs:
  target:
    description: Target path
    required: true
runs:
  using: bash
  args: ["-lc"]
  main: ./main.sh
```

## Fields

### `name`

- Purpose: task package name.

### `description`

- Purpose: package summary.

### `inputs`

- Purpose: named inputs with descriptions, defaults, and required flags.
- Input names should be stable and descriptive.

```yaml
inputs:
  target:
    description: Target path
    required: true
  configuration:
    default: Release
```

### `runs`

- Purpose: execution engine configuration.

#### `runs.using`

- Common values: `docker`, `deno`, `bun`, `composite`, `bash`, `sh`.
- Example: `using: deno`

#### `runs.image`

- Purpose: Docker image when `using: docker`.

#### `runs.args`

- Purpose: extra execution arguments.

#### `runs.main`

- Purpose: main script file for `deno`/`bun` packages.
- Example: `main: mod.ts`

#### `runs.steps`

- Purpose: composite execution steps.
- Each step follows the Cast task shape.

## Notes

- Remote task definitions are experimental and may evolve.
- Inputs are injected as `INPUT_*` environment variables.
- Keep package names and input ids stable so callers can pass `with:` keys predictably.
