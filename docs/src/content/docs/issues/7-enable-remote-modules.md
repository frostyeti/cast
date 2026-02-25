---
title: "7-enable-remote-modules"
type: epic
tags: ["epic"]
---

# Enable Remote Modules


## Implementation Plan

1. **Schema Updates:**
   - Extend the root `castfile` schema to support a `modules` block.
   - Define syntax for remote module references, allowing Git URIs (e.g., `github.com/org/repo@v1.0.0`) and HTTP tarball URLs (e.g., `https://registry.example.com/module-v1.0.0.tar.gz`).

2. **Fetching and Extraction Engine (`internal/modules/fetch.go`):**
   - Implement fetching logic to download remote modules into `.cast/modules/`.
   - **Git Support:** Perform a shallow clone of the specific tag or commit hash.
   - **Tarball Support:** Use standard Go `net/http` and `archive/tar` to download and unpack compressed modules, similar to npm package extraction.
   - Ensure the extracted module directory is keyed by its name and version to support offline use.

3. **Checksum Lifecycle and Resolution (`internal/modules/checksum.go`):**
   - Compute the hash of the current `castfile` and store it locally (e.g., `.cast/checksum.json`).
   - On invocation, if the `castfile` checksum has changed, iterate through the `modules` block.
   - Identify new or version-bumped modules and initiate a fetch for any that are not already present in `.cast/modules/`.

4. **Runtime Integration (`internal/projects/project.go`):**
   - Load the YAML definitions from the downloaded modules.
   - Merge or register their exposed tasks and configurations so they can be executed seamlessly (e.g., treating them as included sub-workspaces or scoped task namespaces).

5. **Update Command (`cmd/update.go`):**
   - Implement a new CLI command `cast modules update` (or a global `cast update`) that allows users to force a refresh of the module cache without running a specific task.