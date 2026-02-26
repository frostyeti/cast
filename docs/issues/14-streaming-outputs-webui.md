# Issue 14: Stream Task/Job Output to Web UI

## Description
When jobs or tasks are running on the Cast server, we need the ability to stream their live standard output (stdout) and standard error (stderr) to connected web clients.

## Requirements
- Implement WebSocket or Server-Sent Events (SSE) endpoints in `internal/web/server.go`.
- Create a broadcaster/buffer mechanism inside the `runstatus` or `projects.RunTask` logic that captures logs and broadcasts them to active WebSocket subscribers.
- Provide historical logs if a user connects to a stream that has already started.

---

