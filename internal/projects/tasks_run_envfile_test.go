package projects_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestRunTask_ClearsCastEnvAfterEachTask(t *testing.T) {
	projectDir := t.TempDir()
	castEnvPath := filepath.Join(projectDir, "cast.env")
	projectFile := filepath.Join(projectDir, "castfile.yaml")

	content := `
name: cast-env-clear
env:
  CAST_ENV: ` + castEnvPath + `
tasks:
  first:
    uses: bash
    run: |
      printf 'FIRST=one\n' > "$CAST_ENV"
  second:
    uses: bash
    run: |
      if [ -s "$CAST_ENV" ]; then
        echo "stale=$(cat "$CAST_ENV")"
        exit 1
      fi
      printf 'SECOND=two\n' > "$CAST_ENV"
  print:
    uses: bash
    run: echo "FIRST=$FIRST SECOND=$SECOND"
`

	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	proj := &projects.Project{}
	if err := proj.LoadFromYaml(projectFile); err != nil {
		t.Fatalf("failed to load project: %v", err)
	}

	var stdout bytes.Buffer
	_, err := proj.RunTask(projects.RunTasksParams{
		Targets:     []string{"first", "second", "print"},
		Context:     context.Background(),
		ContextName: "default",
		Stdout:      &stdout,
		Stderr:      &stdout,
	})
	if err != nil {
		t.Fatalf("failed to run tasks: %v\nOutput: %s", err, stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "FIRST=one SECOND=two") {
		t.Fatalf("expected propagated env output, got: %s", output)
	}

	data, err := os.ReadFile(castEnvPath)
	if err != nil {
		t.Fatalf("failed to read CAST_ENV file: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected CAST_ENV file to be cleared after run, got: %q", string(data))
	}
}
