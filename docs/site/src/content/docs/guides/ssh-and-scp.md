---
title: SSH and SCP Tasks
description: How to use Cast's built-in SSH and SCP runners.
---

# SSH and SCP Tasks

Cast ships with built-in `ssh` and `scp` task runners for remote execution and file transfer.

## SSH tasks

Use `ssh` when you want Cast to run a shell script on remote hosts.

```yaml
tasks:
  deploy:
    uses: ssh
    hosts: [prod]
    run: |
      cd /srv/app
      docker compose up -d
```

### Options

- `hosts`: required inventory targets.
- `run`: remote shell script.
- `with.script`: load the script from a local file before execution.
- `with.max-parallel`: override the default host worker count.
- `with.send-env`: when `true`, forward task env values with SSH `SendEnv`. Default is `false`.

### Script loading

If `with.script` is set, Cast reads that file and uses it as the SSH script.
Relative paths resolve from the task working directory, and absolute paths stay absolute.

```yaml
tasks:
  deploy:
    uses: ssh
    with:
      script: ./scripts/deploy.sh
```

### Template mode

Set `template: gotmpl` to render the SSH script before execution.

By default, task env values are only available to the local template renderer. Cast does not send them to the remote SSH session unless `with.send-env: true` is set.

```yaml
tasks:
  deploy:
    uses: ssh
    template: gotmpl
    env:
      APP_ENV: prod
    run: |
      echo "Deploying {{ .env.APP_ENV }} to {{ .target.Host }}"
```

### Forwarding environment variables

Set `with.send-env: true` if you need the remote SSH session to receive task env values.

```yaml
tasks:
  remote-env:
    uses: ssh
    hosts: [prod]
    env:
      APP_ENV: prod
    with:
      send-env: true
    run: echo "$APP_ENV"
```

Many SSH servers reject `SendEnv` by default unless `AcceptEnv` is configured server-side. If that happens, Cast returns a diagnostic that points to `with.send-env` and the server `AcceptEnv` setting.

```yaml
tasks:
  deploy:
    uses: ssh
    template: gotmpl
    run: |
      echo "Deploying {{ .env.APP_ENV }} to {{ .target.Host }}"
```

## SCP tasks

Use `scp` to copy files to or from remote hosts.

```yaml
tasks:
  sync-assets:
    uses: scp
    hosts: [prod]
    with:
      files:
        - ./dist:/srv/app/dist
```

### Options

- `with.files`: list of `source:destination` pairs.
- `with.direction`: `upload` or `download`.
- `with.max-parallel`: override the default host worker count.

### Paths

- Local upload paths can be relative or absolute.
- Relative paths resolve from the task working directory.
- Remote paths stay remote paths.
- If a path contains `$VAR`, Cast expands it from the task environment.
- Command substitution is disabled for SCP paths.

### Optional files

Prefix or suffix a source or destination path with `?` to make the transfer optional.

```yaml
tasks:
  sync-assets:
    uses: scp
    hosts: [prod]
    with:
      files:
        - ./dist/app.css?:/srv/app/app.css
        - /srv/app/optional.txt?:./optional.txt
```

If the local or remote optional file is missing, Cast skips it.

### Download direction

For downloads, set `with.direction: download`.

```yaml
tasks:
  fetch-logs:
    uses: scp
    hosts: [prod]
    with:
      direction: download
      files:
        - /srv/app/logs/app.log:./logs/app.log
```

### Best practices

- Use absolute paths when you want paths to be unambiguous.
- Use relative paths when the files live inside the project tree.
- Use optional paths for machine-specific or generated files.
