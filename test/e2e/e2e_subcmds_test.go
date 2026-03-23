package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_SubcmdsHelpAndArgPassthrough(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	projectFile := filepath.Join(tmpDir, "castfile")
	content := `
name: subcmd-e2e
subcmds:
  - mysql
tasks:
  mysql:help:
    uses: shell
    help: |
      MYSQL HELP BLOCK
      use: cast mysql test
    run: echo SHOULD_NOT_RUN
  mysql:test:
    uses: shell
    run: echo deno test -A
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	helpCmd := exec.Command("timeout", "15", binPath, "-p", projectFile, "mysql", "--help")
	helpCmd.Dir = tmpDir
	helpOut, err := helpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mysql --help failed: %v\n%s", err, string(helpOut))
	}
	if !strings.Contains(string(helpOut), "MYSQL HELP BLOCK") {
		t.Fatalf("expected subcommand help block, got: %s", string(helpOut))
	}

	runCmd := exec.Command("timeout", "15", binPath, "-p", projectFile, "mysql", "test", "--clean", "./path/to/file")
	runCmd.Dir = tmpDir
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mysql test failed: %v\n%s", err, string(runOut))
	}
	if !strings.Contains(string(runOut), "deno test -A --clean ./path/to/file") {
		t.Fatalf("expected args passthrough for shell command, got: %s", string(runOut))
	}
}
