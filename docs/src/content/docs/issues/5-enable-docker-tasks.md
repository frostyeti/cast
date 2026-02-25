---
title: "5-enable-docker-tasks"
type: feature
tags: ["feature"]
---

# Enable Docker Tasks

## Implementation Plan

1. **Schema Update (`internal/types/task.go` & `schemas/`):**
   - The `Task` definition in `task.go` already handles `use` and `uses`. Update documentation and examples to primarily use the `uses:` property (e.g., `uses: docker`).
   - Support a `with` dictionary containing fields like `image`, `command`, `args`, and `volumes`.

2. **Docker Task Runner (`internal/projects/run_docker_task.go`):**
   - Create an execution handler for Docker tasks.
   - Construct a `docker run --rm` command via `github.com/frostyeti/go/exec` instead of `os/exec` to yield a more usable output result.
   - Automatically mount the current working directory to a standard path inside the container, such as `-v "$PWD:/app"` and set `-w /app` so it behaves like GitHub Actions workspaces.
   - Pass additional `volumes`, `args`, and environment variables correctly.

3. **Image Tracking and Management:**
   - Create a central tracking file (e.g., `~/.cast/docker_images.json`) that logs every unique Docker image used by `cast` tasks.
   - Append to this file during task execution if the image is novel.

4. **Purge Command:**
   - Introduce a new CLI command under the tools submenu, e.g., `cast tools docker purge` (or `cast tool docker purge`).
   - Read the tracking file, iterate through the recorded images, and execute `docker rmi <image>` to reclaim disk space.
   - Clear the file of successfully removed images.