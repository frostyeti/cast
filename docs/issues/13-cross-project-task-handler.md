# Issue 13: Cross-Project Task Handler

## Description
Create a new task handler (similar to `shell`, `deno`, `docker`) that allows dispatching a job or task in another completely different project/castfile.

## Requirements
- Add a new scriptx runner/handler (e.g., `uses: cast`).
- Accept configurations to specify the target project directory or explicitly the target `castfile`.
- Accept configurations for which `task` or `job` to run in the target project.
- Inherit the current Cast context, but execute within the scope of the remote project.
- Stream the output of the dispatched project back to the current execution context.

---

