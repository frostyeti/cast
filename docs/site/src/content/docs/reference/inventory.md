---
title: Inventory Reference
description: Standalone inventory YAML reference.
---

# Inventory

An inventory file defines hosts and reusable host defaults.

## Shapes

```yaml
defaults:
  ssh:
    user: deploy

hosts:
  api:
    host: 10.0.0.10
    defaults: ssh
```

## Fields

### `defaults`

- Purpose: named default host connection blocks.
- Example: `ssh`.

### `hosts`

- Purpose: host entries, either as a map or a sequence.
- Example map form:

```yaml
hosts:
  api:
    host: 10.0.0.10
    user: deploy
```

- Example sequence form:

```yaml
hosts:
  - api.example.com
  - user@worker.example.com:2222
```

## Host syntax

### Scalar host syntax

- `host`
- `host:port`
- `user@host`
- `user@host:port`

### Mapping host syntax

```yaml
hosts:
  api:
    host: 10.0.0.10
    user: deploy
    port: 22
    defaults: ssh
```

## Host defaults

- Fields: `user`, `identity`, `password`, `port`, `agent`, `tags`, `meta`, `os`
- `password` supports environment expansion and optional command substitution when `config.substitution` is enabled
- `identity` supports `~` expansion plus environment expansion
- `agent: true` requires a working SSH agent and fails with a clear diagnostic if `SSH_AUTH_SOCK` is unavailable
- `os` accepts platform metadata and can be used to scope hosts by OS family/version

```yaml
defaults:
  ssh:
    user: deploy
    identity: ~/.ssh/id_ed25519
    tags: [prod, linux]
    agent: false
    os:
      platform: linux
      family: linux
```

## Authentication resolution

- Cast prefers host-specific settings first, then falls back to `CAST_SSH_PASS`, then `SSH_PASS` when no password is set in inventory.
- SSH/SCP task runners can use identity files, passwords, or the current SSH agent.
- When `agent: true` is set on a host, Cast requires the SSH agent instead of silently falling back.

## Notes

- Use `defaults` to avoid repeating credentials and OS metadata.
- Host names and aliases should be readable and stable.
