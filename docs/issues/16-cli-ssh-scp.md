# Issue 16: CLI Tools for SSH & SCP

## Description
Allow users to interact with servers defined in the cast inventory directly from the CLI without needing to look up IP addresses, users, or keys manually.

## Requirements
- Add `cast ssh <host>` command to spawn an interactive shell on an inventory host.
- Add `cast scp <src> <dest>` command (with `--targets` list) to push or pull files across one or multiple hosts.
- Add `--script` flag to let users define a local shell script to execute on the targets.
- Add `--template` flag to use Go templating (`text/template`) on the script, evaluating variables before sending it to the targets for execution.

---

