package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_ContextFlagSetsEnvVar(t *testing.T) {
	// 1. Build the cast binary
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}
	t.Log("Build complete.")

	// 2. Create a temporary project
	t.Log("Creating temp castfile...")
	yamlFile := filepath.Join(tmpDir, "castfile.yaml")
	yamlData := `
name: Context Test Project
tasks:
  print-context:
    uses: shell
    run: |
      echo "CAST_CONTEXT=$CAST_CONTEXT"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	// Test 1: Verify default context is "default" when no flag is provided
	t.Run("default_context_when_no_flag", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "print-context")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=default") {
			t.Errorf("expected CAST_CONTEXT=default, got: %s", string(output))
		}
	})

	// Test 2: Verify -c flag sets CAST_CONTEXT correctly
	t.Run("short_flag_sets_context", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "-c", "staging", "print-context")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=staging") {
			t.Errorf("expected CAST_CONTEXT=staging when using -c staging, got: %s", string(output))
		}
	})

	// Test 3: Verify --context flag also works
	t.Run("long_flag_sets_context", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "--context", "development", "print-context")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=development") {
			t.Errorf("expected CAST_CONTEXT=development when using --context development, got: %s", string(output))
		}
	})

	// Test 4: Verify CAST_CONTEXT env var from environment is respected when no flag
	t.Run("env_var_respected_when_no_flag", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "print-context")
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(), "CAST_CONTEXT=staging")
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=staging") {
			t.Errorf("expected CAST_CONTEXT=staging from env var, got: %s", string(output))
		}
	})

	// Test 5: Verify -c flag overrides CAST_CONTEXT environment variable
	t.Run("flag_overrides_env_var", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "-c", "development", "print-context")
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(), "CAST_CONTEXT=staging")
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=development") {
			t.Errorf("expected CAST_CONTEXT=development (flag should override env), got: %s", string(output))
		}
	})

	// Test 6: Verify 'run' subcommand respects -c flag
	t.Run("run_subcommand_with_flag", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "run", "-c", "production", "print-context")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=production") {
			t.Errorf("expected CAST_CONTEXT=production when using run -c production, got: %s", string(output))
		}
	})

	// Test 6b: Verify env CAST_CONTEXT is respected for run shortcut
	t.Run("run_subcommand_respects_env_context", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "run", "print-context")
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(), "CAST_CONTEXT=dev")
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=dev") {
			t.Errorf("expected CAST_CONTEXT=dev from environment, got: %s", string(output))
		}
	})

	// Test 6c: Verify task run namespace respects env CAST_CONTEXT
	t.Run("task_run_subcommand_respects_env_context", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "task", "run", "print-context")
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(), "CAST_CONTEXT=dev")
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=dev") {
			t.Errorf("expected CAST_CONTEXT=dev from environment for task run, got: %s", string(output))
		}
	})

	// Test 7: Verify 'run' subcommand -c flag overrides CAST_CONTEXT env var
	t.Run("run_subcommand_flag_overrides_env", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "run", "-c", "production", "print-context")
		runCmd.Dir = tmpDir
		runCmd.Env = append(os.Environ(), "CAST_CONTEXT=staging")
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=production") {
			t.Errorf("expected CAST_CONTEXT=production (flag should override env), got: %s", string(output))
		}
	})

	// Test 8: Verify custom context names with special characters
	t.Run("custom_context_name", func(t *testing.T) {
		runCmd := exec.Command("timeout", "5", binPath, "-c", "my-custom-context", "print-context")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CAST_CONTEXT=my-custom-context") {
			t.Errorf("expected CAST_CONTEXT=my-custom-context, got: %s", string(output))
		}
	})

	// Test 9: Verify CAST_CONTEXT is available in task env with variable substitution
	t.Run("context_available_in_task_env", func(t *testing.T) {
		envTestFile := filepath.Join(tmpDir, "castfile-env-test.yaml")
		envTestData := `
name: Env Context Test
tasks:
  check-env:
    uses: shell
    run: echo "CONTEXT_VALUE=${CAST_CONTEXT}"
`
		if err := os.WriteFile(envTestFile, []byte(envTestData), 0644); err != nil {
			t.Fatalf("failed to write castfile: %v", err)
		}

		runCmd := exec.Command("timeout", "5", binPath, "-p", envTestFile, "-c", "test-env", "check-env")
		runCmd.Dir = tmpDir
		output, err := runCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run cast command: %v\n%s", err, string(output))
		}
		if !strings.Contains(string(output), "CONTEXT_VALUE=test-env") {
			t.Errorf("expected CONTEXT_VALUE=test-env, got: %s", string(output))
		}
	})
}
