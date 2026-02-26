# Issue 18: Serve Static Site on `cast web`

## Description
When running the `cast web` command, the web server should serve the embedded Svelte static site (from Issue 17) alongside the API endpoints.

## Requirements
- Update `internal/web/server.go` to use `//go:embed` to include the built UI assets.
- Configure the HTTP multiplexer (e.g., net/http or whatever framework is used) to serve the embedded files at the root `/`.
- Ensure fallback routing for SPA (Single Page Application) so that any non-API route returns `index.html` to allow client-side routing.
- API endpoints should remain under `/api/...` to avoid collisions.
