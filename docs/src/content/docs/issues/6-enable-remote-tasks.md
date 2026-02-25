---
title: "6-enable-remote-tasks"
type: epic
tags: ["epic"]
---

# Enable Remote Tasks

## Implementation Plan

1. **Remote Task Resolution and Caching (`internal/projects/remote.go`):**
   - Update `internal/types/task.go` to support the `use:` property.
   - Parse `use:` values to detect Git URIs (e.g., `github.com/...`) vs JSR/NPM scopes (e.g., `@scope/package`).
   - Download dependencies to a central cache directory (e.g., `.cast/tasks/`).
   - **Git Tasks:** Perform a shallow git clone or download a tarball for the specified tag/version.
   - **JSR Tasks:** Fetch the manifest and module using standard HTTP requests or Deno's tooling.

2. **Deno Wrapper Scripting (`internal/projects/run_deno_task.go`):**
   - When running a Deno task, generate a temporary wrapper script.
   - The wrapper will dynamically import the user's task module.
   - It will orchestrate the lifecycle by checking for and invoking exported `setup()`, `run()`, and `teardown()` functions sequentially.
   - Inject the task's `with:` arguments into `Deno.env` so the module can access them via `Deno.env.get()`.

3. **Checksum Optimization:**
   - Compute a SHA256 checksum of the `castfile` whenever tasks are evaluated.
   - Save this checksum in `.cast/state.json`. 
   - On subsequent runs, if the checksum matches, bypass the network check and execute the local cache. If it differs, iterate over all `use:` directives and re-validate/download missing packages.

4. **Dedicated Update Command:**
   - Add `cast update` or `cast tasks update` in the `cmd` package.
   - Allow users to force-refresh local task caches independently of task execution.

5. **Trust and Security Policies:**
   - Introduce a new block in the `castfile` configuration (e.g., `trusted_sources: ["github.com/org/*"]`).
   - Before downloading any remote task, validate its URI against the trusted source patterns to prevent execution of untrusted scripts.