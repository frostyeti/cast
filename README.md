# Cast

**Cast** is a powerful task runner and automation tool designed to manage multi-step workflows, scripts, remote execution, and project automation. It supports multiple runners natively, allowing you to seamlessly integrate tasks written in Bash, Deno, Node.js, Python, PowerShell, Go, and more.

## Installation

### Linux / macOS (Bash)

You can easily install Cast using the following curl command:

```bash
curl -sL https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.sh | bash
```

### Windows (PowerShell)

To install Cast on Windows, open a PowerShell terminal and run:

```powershell
irm https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.ps1 | iex
```

> **Note:** By default, Cast installs to `~/.local/bin` (on Linux/Mac) or `~/AppData/Local/Programs/bin` (on Windows). You can override this by setting the `CAST_INSTALL_DIR` environment variable before running the installation script.

## Getting Started

Cast uses a `castfile.yaml` (or `cast.yaml`) to define tasks in your repository. 

### 1. Initialize a Project

Create a `castfile.yaml` in the root of your project:

```yaml
name: My Awesome Project
version: 1.0.0

tasks:
  hello:
    desc: Says hello
    uses: bash
    run: echo "Hello from Cast!"

  build:
    desc: Build the project
    uses: deno
    run: |
      console.log("Building with Deno...");
```

### 2. List Tasks

```bash
cast list
```

### 3. Run a Task

```bash
cast run hello
```

## Features & Use Cases

### 🛠 Polyglot Scripting
Cast natively supports multiple scripting environments. You can easily write individual tasks in `bash`, `node`, `bun`, `deno`, `python`, `powershell`, or `go`. The appropriate runner is automatically invoked.

```yaml
tasks:
  clean:
    uses: python
    run: |
      import os
      print("Cleaning up...")
```

### 🌍 Remote Task Execution (SSH)
Run shell commands natively on remote hosts using Cast's built-in SSH capabilities.

```yaml
inventories:
  - ./production.yaml

tasks:
  deploy:
    hosts: [web-server]
    uses: ssh
    run: |
      cd /var/www/app
      git pull
      systemctl restart app
```

### 📦 Modular Imports & Inventories
Projects can be organized across multiple repositories or directories. Import reusable tasks, pipelines, or infrastructure host inventories directly.

```yaml
imports:
  - from: github.com/frostyeti/shared-tasks
    ns: core

inventories:
  - ./infrastructure/staging.yaml

tasks:
  deploy:
    needs: [core:build, core:test]
    run: echo "Deploying..."
```

### 🔄 Dynamic Environments & Dotenv
Inject `.env` files based on dynamic contexts (e.g., `dev`, `staging`, `prod`) to keep secrets safe and configurations portable.

```yaml
dotenv:
  - path: .env
  - path: .env.production
    contexts: [prod]
```

Run with context:
```bash
cast run deploy --context prod
```

### 🛡 Task Fallbacks
If a task script is isolated inside a custom directory, Cast can natively find `.yaml` and `.task` files using fallback directories.

```bash
# Finds the task in .cast/tasks or custom folders!
CAST_TASKS_DIR=my-custom-scripts cast run custom-task
```

## Documentation
For more examples, refer to the schema validations provided in `schemas/castfile.schema.json` and explore module/import features!
