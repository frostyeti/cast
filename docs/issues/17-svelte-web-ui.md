# Issue 17: Embedded Svelte Static Web UI

## Description
Build a modern, interactive Static Site UI using Svelte that gets compiled and embedded directly into the `cast` CLI binary, serving as the frontend for the Cast Web Server.

## Requirements
- **Framework:** Svelte + Vite (or SvelteKit in SPA mode).
- **Location:** Place inside a `ui/` directory.
- **Build Process:** Add a pre-build step (`npm run build`) that outputs static files to a dist directory. Use Go 1.16+ `//go:embed` in `internal/web/` to serve these assets.
- **Features:**
  - View all parsed projects, jobs, and tasks.
  - Trigger/Run jobs and tasks from the UI.
  - Real-time streaming output of running jobs/tasks (using WebSockets from Issue 14).
  - Historical view of past runs and their statuses.
  - Interactive Terminal component (using `xterm.js`) to SSH into inventory servers (using WebSockets from Issue 15).
