package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostyeti/cast/internal/projects"
)

func TestResolveWorkspaceProjectByAlias_StripsAtAndMatchesAlias(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	serviceDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "castfile"), []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write service castfile: %v", err)
	}

	rootContent := "name: root\nworkspace:\n  aliases:\n    bob: services/api\n"
	if err := os.WriteFile(rootFile, []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	project := &projects.Project{}
	if err := project.LoadFromYaml(rootFile); err != nil {
		t.Fatalf("load root project: %v", err)
	}

	entry, err := resolveWorkspaceProjectByAlias(project, "@bob")
	if err != nil {
		t.Fatalf("resolve workspace alias @bob failed: %v", err)
	}

	want := filepath.Join(serviceDir, "castfile")
	if entry.Path != want {
		t.Fatalf("resolved path = %q, want %q", entry.Path, want)
	}
}

func TestResolveWorkspaceProjectByAlias_DuplicateBasenameRequiresPath(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	serviceOneDir := filepath.Join(tmpDir, "apps", "api")
	serviceTwoDir := filepath.Join(tmpDir, "services", "api")

	for _, dir := range []string{serviceOneDir, serviceTwoDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir service dir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "castfile"), []byte("name: api\n"), 0o644); err != nil {
			t.Fatalf("write castfile for %s: %v", dir, err)
		}
	}

	rootContent := "name: root\nworkspace:\n  aliases:\n    one-api: apps/api\n    two-api: services/api\n"
	if err := os.WriteFile(rootFile, []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	project := &projects.Project{}
	if err := project.LoadFromYaml(rootFile); err != nil {
		t.Fatalf("load root project: %v", err)
	}
	if err := project.InitWorkspace(); err != nil {
		t.Fatalf("init workspace: %v", err)
	}

	entry, err := resolveWorkspaceProjectByAlias(project, "@one-api")
	if err != nil {
		t.Fatalf("resolve explicit alias @one-api failed: %v", err)
	}
	if !strings.Contains(entry.Path, filepath.Join("apps", "api", "castfile")) {
		t.Fatalf("expected resolved path to include apps/api/castfile, got %q", entry.Path)
	}

	entry, err = resolveWorkspaceProjectByAlias(project, "@two-api")
	if err != nil {
		t.Fatalf("resolve explicit alias @two-api failed: %v", err)
	}
	if !strings.Contains(entry.Path, filepath.Join("services", "api", "castfile")) {
		t.Fatalf("expected resolved path to include services/api/castfile, got %q", entry.Path)
	}
}

func TestSelfWorkspaceListCommand_ListsAliasesAndPaths(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	serviceDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "castfile"), []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write service castfile: %v", err)
	}

	rootContent := "name: root\nworkspace:\n  aliases:\n    bob: services/api\n"
	if err := os.WriteFile(rootFile, []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "ws", "ls", "-p", rootFile}, "")
	if err != nil {
		t.Fatalf("self ws ls failed: %v", err)
	}

	if !strings.Contains(out, "ALIASES") {
		t.Fatalf("expected aliases section header in output, got: %s", out)
	}
	if !strings.Contains(out, "bob") {
		t.Fatalf("expected bob alias in output, got: %s", out)
	}
	if !strings.Contains(out, filepath.ToSlash(filepath.Join("services", "api", "castfile"))) && !strings.Contains(out, filepath.Join("services", "api", "castfile")) {
		t.Fatalf("expected services/api/castfile path in output, got: %s", out)
	}
}

func TestSelfWorkspaceListCommand_GroupsUnaliasedPathsAtEnd(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	aliasedDir := filepath.Join(tmpDir, "services", "api")
	unaliasedDir := filepath.Join(tmpDir, "apps", "api")

	for _, dir := range []string{aliasedDir, unaliasedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "castfile"), []byte("name: demo\n"), 0o644); err != nil {
			t.Fatalf("write castfile in %s: %v", dir, err)
		}
	}

	rootContent := "name: root\nworkspace:\n  include:\n    - \"**\"\n  aliases:\n    bob: services/api\n"
	if err := os.WriteFile(rootFile, []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	out, err := executeRootForTest([]string{"self", "workspace", "ls", "-p", rootFile}, "")
	if err != nil {
		t.Fatalf("self workspace ls failed: %v", err)
	}

	aliasesIdx := strings.Index(out, "ALIASES")
	pathsIdx := strings.Index(out, "PATHS")
	if aliasesIdx < 0 || pathsIdx < 0 {
		t.Fatalf("expected ALIASES and PATHS sections, got: %s", out)
	}
	if aliasesIdx > pathsIdx {
		t.Fatalf("expected ALIASES before PATHS, got: %s", out)
	}

	if !strings.Contains(out, "bob") {
		t.Fatalf("expected aliased project in ALIASES section, got: %s", out)
	}
	if !strings.Contains(out, filepath.ToSlash(filepath.Join("apps", "api", "castfile"))) && !strings.Contains(out, filepath.Join("apps", "api", "castfile")) {
		t.Fatalf("expected unaliased path in PATHS section, got: %s", out)
	}
}
