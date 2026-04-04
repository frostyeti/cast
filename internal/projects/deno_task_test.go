package projects

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDenoTaskWrapper_InlineScriptIncludesResourceDiagnostic(t *testing.T) {
	tmpDir := t.TempDir()
	wrapperPath, cleanup, err := writeDenoTaskWrapper("console.log('hello from inline deno')", tmpDir, "inline-deno", "inline-deno")
	if err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "hello from inline deno") {
		t.Fatalf("expected wrapped script to include original inline code, got %q", out)
	}
	if !strings.Contains(out, "__castScheduleLingeringResourceDiagnostic") {
		t.Fatalf("expected Deno wrapper diagnostic helper in output, got %q", out)
	}
	if !strings.Contains(out, "Deno.resources()") {
		t.Fatalf("expected Deno resource diagnostic in wrapper, got %q", out)
	}
	if !strings.Contains(out, "Deno.exit(124)") {
		t.Fatalf("expected lingering-resource exit code in wrapper, got %q", out)
	}
}

func TestWriteDenoTaskWrapper_FileScriptPreservesRelativeImports(t *testing.T) {
	tmpDir := t.TempDir()
	helperPath := filepath.Join(tmpDir, "helper.ts")
	if err := os.WriteFile(helperPath, []byte("export const value = 1;\n"), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	scriptPath := filepath.Join(tmpDir, "task.ts")
	if err := os.WriteFile(scriptPath, []byte("import { value } from './helper.ts';\nconsole.log(value);\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	wrapperPath, cleanup, err := writeDenoTaskWrapper(scriptPath, tmpDir, "file-deno", "file-deno")
	if err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	defer cleanup()

	if filepath.Dir(wrapperPath) != tmpDir {
		t.Fatalf("expected wrapper to be created next to source file, got %q", wrapperPath)
	}

	data, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read wrapper: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "import { value } from './helper.ts';") {
		t.Fatalf("expected relative import to be preserved, got %q", out)
	}
	if !strings.Contains(out, "__castScheduleLingeringResourceDiagnostic") {
		t.Fatalf("expected resource diagnostic helper, got %q", out)
	}
}

func TestBuildDenoModuleWrapperIncludesResourceDiagnostic(t *testing.T) {
	wrapper := buildDenoModuleWrapper("file:///tmp/task.ts", `{}`, "module-task")
	if !strings.Contains(wrapper, `import * as mod from "file:///tmp/task.ts";`) {
		t.Fatalf("expected wrapper to import target module, got %q", wrapper)
	}
	if !strings.Contains(wrapper, "__castScheduleLingeringResourceDiagnostic") {
		t.Fatalf("expected resource diagnostic helper, got %q", wrapper)
	}
	if !strings.Contains(wrapper, "Deno.resources()") {
		t.Fatalf("expected Deno resource listing, got %q", wrapper)
	}
	if !strings.Contains(wrapper, "process.exit(1)") {
		t.Fatalf("expected wrapper to preserve failure exit handling, got %q", wrapper)
	}
}

func TestNewDenoLingeringResourceError(t *testing.T) {
	err := newDenoLingeringResourceError("deno-open-resource")
	if err == nil || !strings.Contains(err.Error(), "left Deno resources open") {
		t.Fatalf("expected lingering resource error message, got %v", err)
	}
	if !strings.Contains(err.Error(), "deno-open-resource") {
		t.Fatalf("expected task id in lingering resource error, got %v", err)
	}
}
