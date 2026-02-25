---
tags: ["chore"]
type: chore
---

# Implement tests

## Implementation Plan

1. **Unit Tests:**
   - Write standard Go unit tests (`*_test.go`) alongside the source files in `internal/` and `cmd/` packages.
   - Run via `go test ./...`.

2. **Integration Tests:**
   - Create separate files for integration tests using the `// +build integration` directive.
   - Execute these tests using the command `go test -tags=integration ./...`.
   - Utilize Testcontainers for Go if external dependencies are needed.

3. **End-to-End (E2E) Tests:**
   - Store E2E tests within the `test/e2e` directory.
   - The test suite should automatically compile the `cast` binary.
   - Write the tests so that they build `cast`, generate files and directories as needed, run the tests, and then remove files/folders when done.