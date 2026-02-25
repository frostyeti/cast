---
title: "GitHub Actions Style Remote Tasks"
type: feature
tags: ["feature", "remote-tasks"]
---

# GitHub Actions Style Remote Tasks (`casttask.yaml`)

## Overview

Currently, remote tasks resolve to a Git repository and execute a root `.ts` or `.js` file directly. To enable more powerful, declarative, and reusable tasks, Cast should support a GitHub Actions-style remote task execution model driven by a `casttask.yaml` definition file.

This enhancement will allow remote repositories to define typed task inputs, execution types (Docker, Deno, Composite), and support proper Semantic Versioning for tags and sub-directory targeting.

## Implementation Plan

### 1. Git Semantic Version Tag Resolution
- Upgrade the Git fetching logic in `internal/projects/remote.go` to handle semantic versioning references.
- If a user specifies a target like `uses: github.com/org/repo@v1`, the fetcher should query the remote repository's tags and resolve it to the highest matching `v1.x.x` tag (e.g., `v1.3.3`) rather than blindly attempting to clone a branch named `v1`.
- Support any standard Git provider (GitHub, GitLab, Bitbucket) as long as it exposes Git tags in a standard format.

### 2. Subdirectory Targeting
- Properly parse complex URIs that include paths after the version tag, e.g., `github.com/org/tasks-project@v1.0.1/subfolder`.
- Clone the repository at the correct tag, but evaluate the task execution strictly within the context of the targeted `subfolder/`.

### 3. Task Definition Schema (`casttask.yaml`)
- Look for a `casttask.yaml` (or `.yml`) file in the targeted directory (root or subfolder).
- This YAML file should dictate *how* the remote task runs, supporting three primary execution engines:
  - **Deno/Script:** Specifies a script entrypoint to run via the Deno wrapper.
  - **Docker:** Specifies a `Dockerfile` or pre-built `image` to run, mounting the workspace appropriately.
  - **Composite:** Defines an ordered list of run steps/tasks (mirroring the syntax of standard Castfile tasks) executed sequentially.

### 4. Input Validation and Environment Injection
- **Definition:** The `casttask.yaml` should define an `inputs:` block specifying expected parameters (with optional types or defaults).
- **Validation:** When `cast` runs the task, it must evaluate the `with:` arguments from the caller's `castfile.yaml` at runtime, ensuring all required inputs are met and validating them against the `casttask.yaml` definition.
- **Injection:** Transform the validated parameters into environment variables prefixed with `INPUT_` (e.g., `with: { hello: "world" }` becomes `INPUT_HELLO="world"`) and inject them into the execution context (Deno runtime, Docker container, or Composite step environments) mirroring GitHub Actions behavior.

## Example Usage

**Caller (`castfile.yaml`):**
```yaml
tasks:
  lint:
    uses: "github.com/my-org/cast-actions@v2/linter"
    with:
      strict: "true"
      target_dir: "src/"
```

**Remote Definition (`cast-actions/linter/casttask.yaml`):**
```yaml
name: "Linter Action"
description: "Lints the codebase"
inputs:
  strict:
    description: "Fail on warnings"
    default: "false"
  target_dir:
    description: "Directory to lint"
    required: true

runs:
  using: "docker"
  image: "alpine/eslint:latest"
  args:
    - "--strict=${INPUT_STRICT}"
    - "${INPUT_TARGET_DIR}"
```
