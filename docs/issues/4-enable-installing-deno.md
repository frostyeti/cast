---
type: feature
tags: ["feature"]
---

# Enable Installing Deno and Mise

## Implementation Plan

1. **Add Tool Commands (`cast tool`):**
   - Create a new set of CLI subcommands under `cast tool` in the `cmd` package.
   - Implement `install`, `where`, and `use` subcommands.
   - For `where` and `use` (e.g., `cast tool where node`, `cast tool use node@20`), act as a proxy and pass the arguments directly to `mise`.

2. **Deno Installation (`cast tool install deno`):**
   - Check if `deno` is already accessible in the current `$PATH`.
   - Determine the host OS (Windows vs. macOS/Linux).
   - For Windows: Execute the standard PowerShell installation script (`irm https://deno.land/install.ps1 | iex`).
   - For macOS/Linux: Execute the standard Shell script (`curl -fsSL https://deno.land/x/install/install.sh | sh`).
   - Log the paths and instruct the user on modifying their shell profiles (`~/.bashrc`, `~/.zshrc`, or Windows Environment Variables) if the script doesn't handle it.

3. **Mise Installation (`cast tool install mise`):**
   - Check if `mise` is already accessible.
   - For Unix-like systems, run the official script: `curl https://mise.run | sh`.
   - Provide follow-up instructions for adding `mise activate` into their shell configuration.

4. **Fallback to Mise for Other Tools (`cast tool install <other>`):**
   - For any tool that is not `deno` or `mise` (e.g., `cast tool install node`), proxy the command to `mise install <tool>`.
   - If `mise` is not installed on the system, prompt the user to run `cast tool install mise` first (or auto-install it).

5. **Integration during Remote Task Execution:**
   - Update the task runner (`internal/projects/run_*.go`) for remote tasks.
   - If a task attempts to use `deno` but it's not installed, catch the error and prompt the user to run `cast tool install deno`.