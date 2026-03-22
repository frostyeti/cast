---
title: Workspace Reference
description: Workspace YAML reference.
---

# Workspace

`workspace` controls nested project discovery and aliases.

## Shapes

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

## Fields

### `include`

- Purpose: glob patterns to include while scanning for nested projects.
- Example: `services/**`

### `exclude`

- Purpose: glob patterns to skip during discovery.
- Example: `node_modules/**`

### `aliases`

- Purpose: named workspace entries mapped to paths.
- Example: `backend: services/api`

```yaml
workspace:
  aliases:
    frontend: apps/web
    api: services/api
```

## Notes

- `workspace: true` enables discovery with defaults.
- `workspace: false` disables nested project scanning.
- Alias names should be simple and stable so `cast @alias` stays predictable.
