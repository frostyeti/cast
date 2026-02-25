---
title: Getting Started
description: Learn how to get started with Cast
---

# Getting Started with Cast

Cast is a task runner and automation tool built in Go.

## Installation

Download the binary from the GitHub releases page or install via Homebrew:

```bash
brew install frostyeti/tap/cast
```

## Basic Castfile

Create a `castfile.yaml` in your project root:

```yaml
name: my-project
tasks:
  hello:
    run: echo "Hello World!"
```

Run it using:

```bash
cast hello
```
