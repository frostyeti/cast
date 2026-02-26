# Standalone Inventories and Fallback Tasks

## Standalone Inventories

In addition to importing external tasks via Cast Modules, you can now import standalone Inventory files directly into your project. These inventories contain hosts and default configurations, and will be securely parsed and merged into your project's `inventory`. 

```yaml
# castfile.yaml
inventories:
  - ./prod.yaml
  - dev
```

Cast supports multiple lookup locations to resolve inventories:
1. **Explicit Paths**: Absolute paths or relative paths (e.g., `./prod.yaml`, `../hosts.yaml`).
2. **Local Caching**: The `.cast/inventory/` directory in your current project.
3. **Global Caching**: The `~/.local/share/cast/inventory/` global directory.

If an extension isn't provided (e.g. `dev`), Cast will automatically look for `dev.yaml` and `dev.yml`.

## Fallback Tasks Routing

When a task refers to a handler that hasn't been locally defined via `uses` in `castfile.yaml`, Cast will now seamlessly fall back to look for matching task extensions locally and globally.

Lookup Order:
1. `CAST_TASKS_DIR/` (e.g., `CAST_TASKS_DIR=my-custom-dir`)
2. `.cast/tasks/` (local project fallback cache)
3. `~/.local/share/cast/tasks/` (global fallback cache)

This behaves exactly like implicitly importing the task block if it exists within the task cache!
