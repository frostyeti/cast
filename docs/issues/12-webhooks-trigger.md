# Issue 12: Server Webhooks to Trigger Jobs/Tasks

## Description
The Cast web server should expose HTTP webhook endpoints that allow external services (like GitHub Actions, CI/CD pipelines, or third-party tools) to trigger specific tasks or jobs.

## Requirements
- Add an endpoint to `internal/web/server.go` (e.g., `POST /api/webhooks/trigger`).
- Define a schema in `castfile.yaml` to configure webhook tokens/secrets securely, mapping them to specific tasks or jobs to avoid unauthorized execution.
- Read query parameters or JSON payloads from the webhook and inject them into the triggered task/job as environment variables or arguments.

---

