package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func completionValuesOnly(items []string) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		v, _, _ := strings.Cut(item, "\t")
		values = append(values, v)
	}
	return values
}

func TestProvideProjectCompletion_RespectsProjectFlag(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	childDir := filepath.Join(tmpDir, "services", "api")
	childFile := filepath.Join(childDir, "castfile")

	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}

	rootContent := "name: root\nworkspace:\n  aliases:\n    api: services/api\ntasks:\n  root-task:\n    run: echo root\n"
	if err := os.WriteFile(rootFile, []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	childContent := "name: api\ntasks:\n  child-task:\n    run: echo child\n"
	if err := os.WriteFile(childFile, []byte(childContent), 0o644); err != nil {
		t.Fatalf("write child castfile: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", "@api", "")
	cmd.Flags().StringP("context", "c", "", "")

	completions, _ := provideProjectCompletion(cmd, []string{"--project", "@api"}, "")
	values := completionValuesOnly(completions)

	if !slices.Contains(values, "child-task") {
		t.Fatalf("expected child-task in completions, got %v", values)
	}
	if slices.Contains(values, "root-task") {
		t.Fatalf("did not expect root-task in child project completions, got %v", values)
	}
}

func TestProvideProjectCompletion_RespectsContextFlag(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")

	content := `
name: demo
config:
  context: prod
  contexts: [prod, dev]
tasks:
  deploy:
    run: echo default
  deploy:prod:
    run: echo prod
  deploy:dev:
    run: echo dev
`
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", projectFile, "")
	cmd.Flags().StringP("context", "c", "dev", "")

	completions, _ := provideProjectCompletion(cmd, []string{"--project", projectFile, "--context", "dev"}, "dep")
	values := completionValuesOnly(completions)

	if !slices.Contains(values, "deploy") {
		t.Fatalf("expected deploy completion, got %v", values)
	}
	for _, v := range values {
		if strings.Contains(v, ":") {
			t.Fatalf("did not expect context-suffixed completion, got %v", values)
		}
	}
}

func TestResolveProjectFileFromFlagOrCwdSupportsWorkspaceAlias(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	childDir := filepath.Join(tmpDir, "services", "api")
	childFile := filepath.Join(childDir, "castfile")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.WriteFile(rootFile, []byte("name: root\nworkspace:\n  aliases:\n    api: services/api\n"), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}
	if err := os.WriteFile(childFile, []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write child castfile: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", "@api", "")
	resolved, err := resolveProjectFileFromFlagOrCwd(cmd)
	if err != nil {
		t.Fatalf("resolve project file: %v", err)
	}
	if resolved != childFile {
		t.Fatalf("resolved project file = %q, want %q", resolved, childFile)
	}
}

func TestProvideProjectFlagCompletionIncludesWorkspaceAliases(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "castfile")
	childDir := filepath.Join(tmpDir, "services", "api")
	childFile := filepath.Join(childDir, "castfile")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.WriteFile(rootFile, []byte("name: root\nworkspace:\n  aliases:\n    api: services/api\n"), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}
	if err := os.WriteFile(childFile, []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write child castfile: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	completions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, "@a")
	if !slices.Contains(completions, "@api") {
		t.Fatalf("expected @api completion, got %v", completions)
	}
}

func TestProvideContextFlagCompletionUsesConfigContextsAndDefault(t *testing.T) {
	tmpDir := t.TempDir()
	projectFile := filepath.Join(tmpDir, "castfile")
	content := "name: demo\nconfig:\n  context: prod\n  contexts: [qa, stage, prod]\n"
	if err := os.WriteFile(projectFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write castfile: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", projectFile, "")
	cmd.Flags().StringP("context", "c", "", "")

	completions, _ := provideContextFlagCompletion(cmd, []string{"--project", projectFile}, "")
	for _, want := range []string{"default", "prod", "qa", "stage"} {
		if !slices.Contains(completions, want) {
			t.Fatalf("expected context completion %q, got %v", want, completions)
		}
	}
}
