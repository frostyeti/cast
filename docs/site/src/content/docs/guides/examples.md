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

## Task and Job Inheritance (Extends)

Reduce boilerplate by using `extends` to inherit properties from another task or job:

```yaml
tasks:
  base-build:
    desc: "Base build setup"
    env:
      NODE_ENV: "production"
    run: npm run build

  build-dev:
    extends: base-build
    env:
      NODE_ENV: "development"

jobs:
  base-deploy:
    timeout: 5m
    steps:
      - run: deploy

  deploy-prod:
    extends: base-deploy
    env:
      REGION: "us-east-1"
```

## Server Webhooks

Trigger jobs or tasks via HTTP endpoints when running Cast as a server:

```yaml
on:
  webhooks:
    github-push:
      job: "deploy-pipeline"
      secret: "super-secret" # Verifies X-Hub-Signature-256 header

    custom-trigger:
      task: "sync-data"
      token: "bearer-token-here" # Verifies Authorization: Bearer <token>

jobs:
  deploy-pipeline:
    steps:
      - run: echo "Triggered from GitHub Push!"
      - run: echo "Payload branch is $WEBHOOK_PAYLOAD_BRANCH"

tasks:
  sync-data:
    run: echo "Sync triggered via webhook!"
```
