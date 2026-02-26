package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_RunJobDownstream(t *testing.T) {
	t.Log("Building cast binary...")
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "cast")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../main.go")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cast binary: %v\n%s", err, string(output))
	}

	t.Log("Creating temp castfile with jobs and dependencies...")
	yamlFile := filepath.Join(tmpDir, "castfile")
	yamlData := `
name: Job Run Test
id: job-run-test
tasks:
  taskA:
    uses: bash
    run: echo "task A executed"
  taskB:
    uses: bash
    run: echo "task B executed"
  taskC:
    uses: bash
    run: echo "task C executed"
jobs:
  jobA:
    steps:
      - "taskA"
  jobB:
    needs: jobA
    steps:
      - "taskB"
  jobC:
    needs: jobB
    steps:
      - "taskC"
  jobIsolated:
    steps:
      - "taskA"
`
	if err := os.WriteFile(yamlFile, []byte(yamlData), 0644); err != nil {
		t.Fatalf("failed to write castfile: %v", err)
	}

	runCmd := exec.Command(binPath, "run", "--job", "jobA")
	runCmd.Dir = tmpDir

	output, _ := runCmd.CombinedOutput()
	outStr := string(output)
	if !strings.Contains(outStr, "task A executed") {
		t.Errorf("missing task A output. Output:\n%s", outStr)
	}
	if !strings.Contains(outStr, "task B executed") {
		t.Errorf("missing task B output. Output:\n%s", outStr)
	}
	if !strings.Contains(outStr, "task C executed") {
		t.Errorf("missing task C output. Output:\n%s", outStr)
	}

	// Try jobB
	runCmd2 := exec.Command(binPath, "run", "--job", "jobB")
	runCmd2.Dir = tmpDir

	output2, _ := runCmd2.CombinedOutput()
	outStr2 := string(output2)
	if strings.Contains(outStr2, "task A executed") {
		t.Errorf("task A should not have run when triggering job B. Output:\n%s", outStr2)
	}
	if !strings.Contains(outStr2, "task B executed") {
		t.Errorf("missing task B output. Output:\n%s", outStr2)
	}
	if !strings.Contains(outStr2, "task C executed") {
		t.Errorf("missing task C output. Output:\n%s", outStr2)
	}
}
