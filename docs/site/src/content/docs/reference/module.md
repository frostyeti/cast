---
title: Cast Module Reference
description: Experimental reusable module YAML reference.
---

# Cast Module

`cast.module` is an experimental reusable module format.

## Shapes

```yaml
id: shared
name: Shared Module
imports:
  - from: github.com/acme/cast-shared
tasks:
  lint:
    run: npm run lint
```

## Fields

### `id`

- Purpose: stable module id.
- Example: `shared`

### `name`

- Purpose: display label for the module.

### `version`

- Purpose: module version string.

### `description` / `desc`

- Purpose: human-readable module summary.

### `imports`

- Purpose: build on other modules or task packs.
- Shapes: string shorthand or object form.

```yaml
imports:
  - github.com/acme/shared
  - from: github.com/acme/shared
    ns: shared
    tasks: [lint, test]
```

### `env`

- Purpose: module-scoped environment variables.

### `dotenv`

- Purpose: module dotenv files.

### `paths`

- Purpose: PATH entries provided by the module.

### `meta`

- Purpose: arbitrary module metadata.

### `tasks`

- Purpose: tasks exported by the module.
- Task keys follow the same naming rules as project tasks.

### `inventory`

- Purpose: module-local inventory defaults and hosts.

## Notes

- Modules are experimental.
- Module-level tasks merge into the project task map.
- Keep exported task ids stable and descriptive.
