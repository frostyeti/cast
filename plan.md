# Plan: Job/Task Run History and Cron Expressions

## 1. Schema & Engine Updates
- Add `Cron *string` field to `types.Job`.
- Parse `cron` key in `types.Job` YAML unmarshaling.

## 2. Server Scheduler Update
- In `internal/web/server.go` -> `scheduleJobs()`:
  - Keep project-level `On.Schedule.Crons` triggering the `default` job.
  - Iterate over all jobs in the project. If `Job.Cron != nil`, ensure `Job.Needs == nil` (log error if violated). Schedule this job on its cron schedule using `s.runJob(p.Schema.Id, job.Id, nil)`.

## 3. Database Updates (`internal/web/db.go`)
- Create a new table `runs` (we can drop `job_runs` or just ignore it) with:
  - `id`, `project_id`, `type` (job/task), `target_id` (job/task ID), `status`, `logs`, `error`, `created_at`, `completed_at`.
- Rename `JobRun` struct to `Run`.
- Update CRUD functions: `insertRun`, `updateRun`, `getRunLogs`, `getRuns(projectID, targetType, targetID)`.

## 4. API Endpoint Updates
- `handleTriggerTask`: Add `Run` record insertion, update on completion, and stream saving, just like `runJob`.
- Refactor `runJob` to `runTarget(projectID, targetType, targetID, env)` maybe, or just keep them separate but both use `insertRun`.
- Update `handleGetJobRuns` to `handleGetRuns` taking `type=job|task` and `targetId`. 

## 5. UI Updates
- **Projects View**: 
  - Add a "Cron: <expr>" badge to Job cards if `cron` is present.
  - Add a "History" button to Job and Task cards.
  - Clicking "History" opens a dialog or side-panel showing a table of previous runs (Status, Start Time, Duration).
  - Clicking a run in the history table opens the logs overlay (if active, streams; if done, fetches full logs from DB via a new API or reusing stream API for historical).
