---
title: Inventories
description: How to define inventory files and inline inventory blocks.
---

# Inventories

Inventories describe remote hosts, connection settings, and reusable defaults.

## Inline inventory in a castfile

Define inventory directly in the project file when the host list is small or local to the project.

```yaml
inventory:
  defaults:
    ssh:
      user: deploy
      port: 22
  hosts:
    web-1:
      host: 10.0.0.10
      defaults: ssh
```

## Standalone inventory files

Use a separate inventory file when you want to share hosts across projects or keep the castfile smaller.

```yaml
defaults:
  ssh:
    user: deploy

hosts:
  api:
    host: 10.0.0.10
    defaults: ssh
```

Then reference it from the project:

```yaml
inventories:
  - ./inventory.yaml
```

You can also use short includes that point at files in `.cast/inventory/`:

```yaml
inventories:
  - prod
  - staging
```

Cast resolves those names as `.cast/inventory/prod.yaml`, `.cast/inventory/prod.yml`, then the same patterns in your user inventory directory. That lets a castfile stay short while still keeping inventory files separate.

## Inventory in modules

Modules can bundle inventory defaults and hosts for reuse.

```yaml
id: shared
name: Shared Module
inventory:
  defaults:
    ssh:
      user: deploy
  hosts:
    worker:
      host: 10.0.0.20
      defaults: ssh
```

Imported module hosts are merged into the project inventory.

## Host forms

Cast accepts a few host shapes:

- `host`
- `host:port`
- `user@host`
- `user@host:port`
- mapping entries with `host`, `user`, `port`, `identity`, `password`, `defaults`, `groups`, or `meta`

## Defaults

Use `defaults` to avoid repeating credentials or metadata.

```yaml
defaults:
  ssh:
    user: deploy
    identity: ~/.ssh/id_ed25519

hosts:
  app:
    host: app.example.com
    defaults: ssh
```

## Paths and interpolation

Inventory file paths are resolved relative to the castfile, and inventory values support the same interpolation rules as other Cast config values.

## Tips

- Keep host names stable and readable.
- Use inline inventories for small projects.
- Use standalone inventory files when you want reuse or separation.
- Use modules when multiple projects should share the same host groups.
