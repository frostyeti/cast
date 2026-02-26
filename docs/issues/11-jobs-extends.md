# Issue 11: Enable `extends` for Jobs

## Description
Currently, tasks can inherit from other tasks using the `extends` keyword. We need to implement this exact same functionality for `jobs`. 

## Requirements
- Add `Extends *string` to the `Job` struct in `internal/types/job.go`.
- In `internal/projects/project.go` (or wherever jobs are initialized/flattened), implement the logic to inherit the base job's properties (Tasks, Env, If, Needs, etc.).
- Ensure that extending a job correctly deep-merges environments and correctly overrides overlapping lists (like tasks).

## Documentation
- Document the `extends` property for both **Tasks** and **Jobs** in the `README.md`.
- Add examples of `extends` in the `docs/site/` documentation.

---

