# Cast

**Cast** is a powerful task runner and automation tool designed to manage multi-step workflows, scripts, remote execution, and project automation. It supports multiple runners natively, allowing you to seamlessly integrate tasks written in Bash, Deno, Node.js, Python, PowerShell, Go, and more.

## Installation

### Linux / macOS

```bash
curl -sL https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/frostyeti/cast/master/eng/scripts/install.ps1 | iex
```

> **Note:** By default, Cast installs to `~/.local/bin` (Linux/Mac) or `~/AppData/Local/Programs/bin` (Windows). Override this by setting the `CAST_INSTALL_DIR` environment variable before running the script.

---

## Autocompletion

Cast supports shell autocompletion for tasks, jobs, and workspace projects!

To enable autocompletion in your shell:

### Bash
```bash
# Enable for the current session
source <(cast completion bash)

# Make permanent (Linux)
echo 'source <(cast completion bash)' >> ~/.bashrc

# Make permanent (macOS)
echo 'source <(cast completion bash)' >> ~/.bash_profile
```

### Zsh
```zsh
# Enable for the current session
source <(cast completion zsh)

# Make permanent
echo 'source <(cast completion zsh)' >> ~/.zshrc
```

### Fish
```fish
# Enable for the current session
cast completion fish | source

# Make permanent
cast completion fish > ~/.config/fish/completions/cast.fish
```

### PowerShell
```powershell
# Enable for the current session
cast completion powershell | Out-String | Invoke-Expression

# Make permanent: Add the following line to your PowerShell profile (find it via $PROFILE)
Invoke-Expression (&cast completion powershell)
```

---

## Getting Started

Cast uses a `castfile.yaml` (or `cast.yaml`) to define tasks in your repository. Create one in your project root:

```yaml
name: My Awesome Project
version: 1.0.0

tasks:
  hello:
    desc: Says hello
    uses: bash
    run: echo "Hello from Cast!"
```

List tasks with `cast list` and run them with `cast run hello`.

---

## Task Configuration

### Dependencies (`needs`)

Chain tasks together using the `needs` array to guarantee tasks execute in order:

```yaml
tasks:
  setup:
    run: echo "Setting up..."
  build:
    needs: [setup]
    run: echo "Building..."
```

### Environment Variables & Secrets

Environment variables can be defined globally or on specific tasks. Cast supports **variable interpolation** and **command substitution** (e.g., retrieving secrets dynamically).

```yaml
env:
  # Simple variable
  PROJECT_ENV: development
  # Command substitution (e.g., fetch a secret)
  DB_PASSWORD: $(aws secretsmanager get-secret-value --secret-id db-pass --query SecretString --output text)
  # Variable interpolation
  API_URL: "https://api.${PROJECT_ENV}.example.com"

tasks:
  deploy:
    env:
      TASK_SPECIFIC_VAR: "Hello"
    run: echo "Deploying with $DB_PASSWORD to $API_URL"
```

### Dotenv Files

You can load environment variables from `.env` files. If a file might not exist, prefix it with `?` to gracefully ignore missing files instead of throwing an error.

```yaml
dotenv:
  - path: .env
  - path: ?.env.local  # Optional file
```

You can also specify `dotenv` configs directly on a task:

```yaml
tasks:
  start:
    dotenv: ["?.env.dev"]
    run: node server.js
```

### CWD and Conditionals (`if`)

Control where a task runs using `cwd`. Control *whether* a task runs using `if` statements. Cast evaluates `if` statements using the [expr library](https://github.com/antonmedv/expr).

```yaml
tasks:
  publish:
    cwd: ./build
    if: env.BRANCH == 'main'
    run: npm publish
```

---

## Extends (Inheritance)

Reduce duplication by using the `extends` keyword to inherit properties from another task or job. The child task or job will deeply merge its properties with the base, allowing you to easily override specific settings like `env`, `run`, `cwd`, and `steps`.

### Task Extends

```yaml
tasks:
  base-build:
    desc: "Base build configuration"
    cwd: ./src
    env:
      BUILD_ENV: "production"
    run: npm run build

  build-dev:
    extends: base-build
    desc: "Build for development"
    env:
      BUILD_ENV: "development"
```

### Job Extends

```yaml
jobs:
  base-deploy:
    desc: "Base deployment job"
    timeout: "5m"
    env:
      REGION: "us-east-1"
    steps:
      - run: deploy

  deploy-eu:
    extends: base-deploy
    desc: "Deploy to EU region"
    env:
      REGION: "eu-west-1"
```

---

## Hooks

Wrap tasks with `before` and `after` tasks using hooks. This is highly useful for setup and teardown processes!

```yaml
tasks:
  pre-flight:
    run: echo "Preparing..."
  post-flight:
    run: echo "Cleaning up..."
  
  deploy:
    hooks:
      before: [pre-flight]
      after: [post-flight]
    run: echo "Deploying..."
```

---

## Runners and Workloads

Cast determines how to execute a script using the `uses` property.

### Polyglot Scripts & Relative Paths

You can use standard language runners (`bash`, `node`, `deno`, `python`, `pwsh`), or point to a local script directly:

```yaml
tasks:
  clean:
    uses: python
    run: |
      import os
      print("Cleaning up...")
      
  script-task:
    uses: ./scripts/custom-runner.sh
    run: echo "Passed to custom runner"
```

### Docker Image Tasks

Easily run workloads inside a Docker container:

```yaml
tasks:
  test-in-docker:
    uses: docker://golang:1.21
    run: go test ./...
```

### Cross-Project Tasks

You can trigger a task or job inside another completely separate `castfile` using the `cast` handler. This is useful for building monorepos or dispatching sub-projects where you don't necessarily want to use the unified Workspace feature.

```yaml
tasks:
  build-frontend:
    uses: cast
    with:
      dir: ./frontend  # Path to the directory containing a castfile
      task: build      # The task to run in that project
      
  deploy-backend:
    uses: cast
    with:
      file: ./backend/castfile.production.yaml
      job: deploy      # You can also trigger a full job
```

---

## Advanced Task Types

### Template Task

Render text dynamically using the `template` property:

```yaml
tasks:
  gen-config:
    uses: template
    template: |
      Server={{ env.SERVER_NAME }}
      Port={{ env.PORT }}
```

### SSH & SCP Tasks

Run shell commands on remote hosts using Cast's built-in SSH capabilities. Cast supports **Go text templates** inside the `run` block for SSH tasks, allowing you to interpolate variables dynamically into the templated script before it is executed remotely.

```yaml
inventories:
  - ./production.yaml

tasks:
  deploy:
    hosts: [web-server]
    uses: ssh
    run: |
      # This is rendered locally before executing on the remote host!
      echo "Deploying to {{ .Host.Host }} as {{ .Host.User }}"
      cd /var/www/app
      docker-compose up -d

  copy-config:
    uses: scp
    hosts: [web-server]
    with:
      src: ./config.yml
      dest: /etc/app/config.yml
```

### Remote Tasks via YAML Imports

You can import tasks, pipelines, or infrastructure configurations from remote Git repositories directly:

```yaml
imports:
  - from: github.com/frostyeti/shared-tasks
    ns: core

tasks:
  deploy:
    needs: [core:build]
    run: echo "Deploying..."
```

---

## Contexts

You can use contexts to conditionally load environment files or target entirely different tasks based on the environment (e.g., `prod`, `dev`).

```yaml
dotenv:
  - path: .env
  - path: .env.production
    contexts: [prod]

tasks:
  "deploy:prod":
    run: echo "Production Deploy"
  
  "deploy:dev":
    run: echo "Dev Deploy"
```

Run with context:
```bash
cast run deploy -c prod
```
*(Cast automatically routes `deploy` to `deploy:prod` based on the context!)*

---

## Workspaces (Nested Projects)

For monorepos or projects with nested components (like a `docker-compose` setup), you can organize sub-projects using Workspaces.

```yaml
workspace:
  include:
    - "services/**"
```

Run a task specifically for a child project using the `@` shortcut without needing to `cd` into the directory:

```bash
cast @backend build
cast @frontend deploy -c prod
```

---

## Jobs

Jobs allow you to group tasks into complex CI/CD-style pipelines and execute them alongside their downstream dependents. 

```yaml
jobs:
  build-all:
    steps:
      - run: build
  
  deploy-all:
    needs: [build-all]
    steps:
      - run: deploy
```

Execute a job and all its downstream jobs:
```bash
cast run --job build-all
```

---

## Server Mode & Webhooks

Cast can run as a long-lived server to execute background tasks on a cron schedule or respond to webhooks.

Start the server:
```bash
cast serve
```

### Webhooks

You can configure webhooks to trigger tasks or jobs remotely (e.g. from GitHub Actions).

```yaml
on:
  webhooks:
    deploy-hook:
      job: deploy-all
      secret: "my-github-secret-key"

jobs:
  deploy-all:
    steps:
      - run: echo "Deploying!"
```

With the server running, you can hit the webhook endpoint:
```bash
curl -X POST http://localhost:8080/api/webhooks/deploy-hook \
  -H "X-Hub-Signature-256: sha256=..." \
  -d '{"branch": "main"}'
```

Query parameters and JSON payload properties are automatically injected into the task environment as `WEBHOOK_QUERY_*` and `WEBHOOK_PAYLOAD_*`.

---

## Exec Command

The `exec` command allows you to run ad-hoc shell commands wrapped in your `castfile` environment. This is exceptionally useful for dynamically pulling secrets or variables defined in your project and using them on the fly.

```bash
# Injects castfile environments, including command-substituted secrets!
cast exec -- psql -U $DB_USER -h $DB_HOST
```

---

## Environment Variables Reference

Cast uses several environment variables to control its behavior and share state between tasks.

### CLI Configuration

| Variable | Description |
|----------|-------------|
| `CAST_PROJECT` | Default project file path (used by `-p` flag) |
| `CAST_CONTEXT` | Default context name (used by `-c` flag) |

### Task Runtime (available within tasks)

| Variable | Description |
|----------|-------------|
| `CAST_ENV` | Path to a temp file. Write `KEY=value` here to export env vars to subsequent tasks. |
| `CAST_PATH` | Path to a temp file. Write a directory path here to prepend it to the `$PATH` of subsequent tasks. |
| `CAST_OUTPUTS` | Path to a temp file for sharing outputs between tasks. |

#### Using CAST_ENV for Task Communication
```yaml
tasks:
  load-env:
    run: echo "API_KEY=secret_123" >> $CAST_ENV

  get-env:
    needs: [load-env]
    run: echo "API_KEY from env: $API_KEY"
```

#### Using CAST_PATH for PATH Updates
```yaml
tasks:
  setup-path:
    run: echo "/custom/bin" >> $CAST_PATH

  use-custom-tool:
    needs: [setup-path]
    run: custom-tool --version
```

### Imported Module Variables
When tasks are executed from imported remote modules, Cast injects the following variables so the task knows where it's running:

| Variable | Description |
|----------|-------------|
| `CAST_FILE` | Path to the module's castfile |
| `CAST_DIR` | Directory containing the module's castfile |
| `CAST_PARENT_FILE` | Path to the parent project's castfile |
| `CAST_PARENT_DIR` | Directory containing the parent project's castfile |
| `CAST_MODULE_ID` | Module's unique identifier |
| `CAST_MODULE_NAME` | Module's name |
