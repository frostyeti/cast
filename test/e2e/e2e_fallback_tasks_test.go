package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackTasksE2E(t *testing.T) {
	tmpDir := t.TempDir()

	castBin := filepath.Join(tmpDir, "cast")
	cmd := exec.Command("go", "build", "-o", castBin, "../../main.go")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build cast binary: %s", string(out))

	projectDir := filepath.Join(tmpDir, "project")
	err = os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// 1. Local fallback task (.cast/tasks/local-task/cast.task)
	localTaskDir := filepath.Join(projectDir, ".cast", "tasks", "local-task")
	err = os.MkdirAll(localTaskDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(localTaskDir, "cast.task"), []byte(`
name: local-task
runs:
  using: deno
  main: mod.ts
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(localTaskDir, "mod.ts"), []byte(`
export function run() {
	console.log("HELLO FROM LOCAL TASK");
}
`), 0644)
	require.NoError(t, err)

	// 2. Custom CAST_TASKS_DIR task
	customTaskDir := filepath.Join(projectDir, "my-custom-dir")
	err = os.MkdirAll(customTaskDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(customTaskDir, "custom-task.yaml"), []byte(`
name: custom-task
runs:
  using: deno
  main: mod.ts
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(customTaskDir, "mod.ts"), []byte(`
export function run() {
	console.log("HELLO FROM CUSTOM TASK");
}
`), 0644)
	require.NoError(t, err)

	// 3. Global task (~/.local/share/cast/tasks/global-task)
	homeDir := filepath.Join(tmpDir, "home")
	globalTaskDir := filepath.Join(homeDir, ".local", "share", "cast", "tasks", "global-task")
	err = os.MkdirAll(globalTaskDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(globalTaskDir, "cast.task"), []byte(`
name: global-task
runs:
  using: deno
  main: mod.ts
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(globalTaskDir, "mod.ts"), []byte(`
export function run() {
	console.log("HELLO FROM GLOBAL TASK");
}
`), 0644)
	require.NoError(t, err)

	// 4. Relative path task
	relTaskDir := filepath.Join(projectDir, "rel-dir")
	err = os.MkdirAll(relTaskDir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(relTaskDir, "rel.task"), []byte(`
name: rel-task
runs:
  using: deno
  main: mod.ts
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(relTaskDir, "mod.ts"), []byte(`
export function run() {
	console.log("HELLO FROM REL TASK");
}
`), 0644)
	require.NoError(t, err)

	// Write castfile
	err = os.WriteFile(filepath.Join(projectDir, "castfile"), []byte(`
tasks:
  run-local:
    uses: local-task
  run-custom:
    uses: custom-task
  run-global:
    uses: global-task
  run-rel:
    uses: ./rel-dir/rel.task
`), 0644)
	require.NoError(t, err)

	t.Run("Local Task", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "run-local")
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(), "HOME="+homeDir)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		assert.Contains(t, string(out), "HELLO FROM LOCAL TASK")
	})

	t.Run("Custom Task Dir", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "run-custom")
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(), "HOME="+homeDir, "CAST_TASKS_DIR=my-custom-dir")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		assert.Contains(t, string(out), "HELLO FROM CUSTOM TASK")
	})

	t.Run("Global Task", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "run-global")
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(), "HOME="+homeDir)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		assert.Contains(t, string(out), "HELLO FROM GLOBAL TASK")
	})

	t.Run("Relative Task", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "run-rel")
		cmd.Dir = projectDir
		cmd.Env = append(os.Environ(), "HOME="+homeDir)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		assert.Contains(t, string(out), "HELLO FROM REL TASK")
	})
}
