---
title: CLI Reference
description: Cast CLI Commands
---

# CLI Reference

## Core Commands

- `cast <task>`: Runs a specific task defined in the `castfile.yaml`.
- `cast update`: Refreshes local task and module caches (clears `.cast/tasks` and `.cast/modules`).

## Tools

Cast provides built-in tooling proxies:

- `cast tool install deno`: Installs Deno via `mise` under the hood.
- `cast tool docker purge`: Purges all cached docker images used by Cast tasks.
