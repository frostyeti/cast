package projects

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/runstatus"
	"github.com/frostyeti/cast/internal/types"
)

func TestRunRemoteTaskSpellComposite(t *testing.T) {
	tmpDir := t.TempDir()
	spellDir := filepath.Join(tmpDir, "examples", "hello-world")
	if err := os.MkdirAll(spellDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	spellPath := filepath.Join(spellDir, "spell")
	spellYaml := "" +
		"name: examples/hello-world\n" +
		"description: runs hello world\n" +
		"runs:\n" +
		"  using: composite\n" +
		"  steps:\n" +
		"    - uses: bash\n" +
		"      run: echo hello world\n"
	if err := os.WriteFile(spellPath, []byte(spellYaml), 0o644); err != nil {
		t.Fatalf("write spell: %v", err)
	}

	if !isCastTaskDefinitionFile(spellPath) {
		t.Fatalf("expected %s to be detected as a cast task definition", spellPath)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	res := runRemoteTask(TaskContext{
		Project: &Project{Dir: tmpDir, Schema: types.Project{}},
		Task: &Task{
			Id:   "ello",
			Uses: spellPath,
			Env:  map[string]string{},
			With: map[string]any{},
		},
		Context: context.Background(),
		Stdout:  stdout,
		Stderr:  stderr,
	})

	if res.Status != runstatus.Ok || res.Err != nil {
		t.Fatalf("expected spell composite to succeed, got status=%s err=%v stderr=%s", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "hello world") {
		t.Fatalf("expected output to contain hello world, got: %s", stdout.String())
	}
}
