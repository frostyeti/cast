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

- Fields: `user`, `identity`, `password`, `port`, `groups`, `meta`, `os`
- `groups` is the YAML name used by the parser, while JSON keeps `tags`
- `os` accepts platform metadata and can be used to scope hosts by OS family/version

```yaml
defaults:
  ssh:
    user: deploy
    identity: ~/.ssh/id_ed25519
    groups: [prod, linux]
    os:
      platform: linux
      family: linux
```

## Notes

- Use `defaults` to avoid repeating credentials and OS metadata.
- Host names and aliases should be readable and stable.
