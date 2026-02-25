---
tags: ["bug"]
type: bug
---

# Bug Fix Workspaces

## Implementation Plan

1. **Fix Empty/Missing Workspace Nodes:**
   - In `internal/types/project.go` or `internal/projects/project.go`, review the YAML unmarshaling logic for the `workspace` node.
   - If `workspace` is missing or `workspace/include` is nil, provide a graceful default fallback instead of throwing an error or skipping workspace evaluations.
   - Ensure the project struct correctly instantiates with empty default sets for includes.

2. **Fix Nested Execution Path:**
   - Modify the initial project discovery process (e.g., inside `cmd/root.go` or `internal/projects/project.go`).
   - Allow the tool to search upwards (parent directories) to find the root `castfile` containing a `workspace` definition.
   - When executed inside a nested directory (e.g., `apps/traefik`):
     - Identify the root workspace path.
     - Resolve the alias of the current subdirectory (e.g., mapped to `@traefik`).
     - Automatically execute the command context as if `cast @traefik <cmd>` was run from the root, so nested `castfile` definitions function correctly within the larger workspace boundary.