---
title: Use Cases & Examples
description: Comprehensive examples for Cast tasks
---

# Examples

## Docker Tasks

Run tasks inside isolated docker containers:

```yaml
tasks:
  build:
    uses: docker
    with:
      image: node:20
    run: npm install && npm run build
```

## Remote Tasks

Execute tasks fetched remotely:

```yaml
trusted_sources:
  - "jsr:"
  - "github.com/org/*"

tasks:
  format:
    uses: "jsr:@std/fmt/colors"
```

## Remote Modules

Import reusable modules from remote sources:

```yaml
modules:
  - from: "github.com/frostyeti/cast-std@v1.0.0"
    ns: "std"

tasks:
  my-task:
    run: cast std:build
```
