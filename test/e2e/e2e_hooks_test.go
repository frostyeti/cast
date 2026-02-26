package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHooksE2E(t *testing.T) {
	tmpDir := t.TempDir()

	castBin := filepath.Join(tmpDir, "cast")
	cmd := exec.Command("go", "build", "-o", castBin, "../../main.go")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build cast binary: %s", string(out))

	// Create a temporary directory for the project
	projectDir := filepath.Join(tmpDir, "hooks-test")
	err = os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	// Write a castfile with hooks: true
	castfileContent := `
tasks:
  build:before:
    uses: shell
    run: echo "running before hook"
  build:after:
    uses: shell
    run: echo "running after hook"
  build:
    hooks: true
    uses: shell
    run: echo "running build task"
  
  # Also test explicit hooks
  deploy:before_deploy:
    uses: shell
    run: echo "running before deploy"
  deploy:after_deploy:
    uses: shell
    run: echo "running after deploy"
  deploy:
    hooks:
      before: before_deploy
      after: after_deploy
    uses: shell
    run: echo "running deploy task"
`
	err = os.WriteFile(filepath.Join(projectDir, "castfile.yaml"), []byte(castfileContent), 0644)
	require.NoError(t, err)

	t.Run("hooks: true runs before and after tasks", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "build")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cast run build failed: %s", string(out))

		output := string(out)

		// Verify all tasks ran in correct order
		require.Contains(t, output, "running before hook")
		require.Contains(t, output, "running build task")
		require.Contains(t, output, "running after hook")

		// Verify order exactly
		beforeIdx := strings.Index(output, "running before hook")
		buildIdx := strings.Index(output, "running build task")
		afterIdx := strings.Index(output, "running after hook")

		require.True(t, beforeIdx < buildIdx, "before hook should run before main task")
		require.True(t, buildIdx < afterIdx, "main task should run before after hook")
	})

	t.Run("explicit hooks run correctly", func(t *testing.T) {
		cmd := exec.Command(castBin, "run", "deploy")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cast run deploy failed: %s", string(out))

		output := string(out)

		// Verify all tasks ran in correct order
		require.Contains(t, output, "running before deploy")
		require.Contains(t, output, "running deploy task")
		require.Contains(t, output, "running after deploy")

		// Verify order exactly
		beforeIdx := strings.Index(output, "running before deploy")
		deployIdx := strings.Index(output, "running deploy task")
		afterIdx := strings.Index(output, "running after deploy")

		require.True(t, beforeIdx < deployIdx, "before hook should run before main task")
		require.True(t, deployIdx < afterIdx, "main task should run before after hook")
	})
}
