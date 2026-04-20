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

func TestProvideProjectFlagCompletion_FiltersToProjectFileNames(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml", "cast", "cast.yaml", "cast.yml", "README.md", "random.txt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("name: demo\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	completions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, "")
	values := completionValuesOnly(completions)

	for _, want := range []string{"castfile", ".castfile", "castfile.yaml"} {
		if !slices.Contains(values, want) {
			t.Fatalf("expected project file completion %q, got %v", want, values)
		}
	}

	for _, unwanted := range []string{"castfile.yml", "cast", "cast.yaml", "cast.yml", "README.md", "random.txt"} {
		if slices.Contains(values, unwanted) {
			t.Fatalf("did not expect non-project file %q in completions: %v", unwanted, values)
		}
	}
}

func TestProvideProjectFlagCompletion_RespectsPathPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "services", "api")

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "castfile.yaml"), []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write castfile.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "castfile.yml"), []byte("name: api-yml\n"), 0o644); err != nil {
		t.Fatalf("write castfile.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "notes.txt"), []byte("ignore\n"), 0o644); err != nil {
		t.Fatalf("write notes.txt: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	completions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, "services/api/")
	values := completionValuesOnly(completions)

	if !slices.Contains(values, "services/api/castfile.yaml") {
		t.Fatalf("expected services/api/castfile.yaml in completions, got %v", values)
	}
	if slices.Contains(values, "services/api/castfile.yml") {
		t.Fatalf("did not expect services/api/castfile.yml in completions, got %v", values)
	}
	if slices.Contains(values, "services/api/notes.txt") {
		t.Fatalf("did not expect services/api/notes.txt in completions, got %v", values)
	}
}

func TestProvideProjectFlagCompletion_DotDirectoryRequiresProjectFiles(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(tmpDir, "with-project"), 0o755); err != nil {
		t.Fatalf("mkdir with-project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "with-project", "castfile"), []byte("name: with-project\n"), 0o644); err != nil {
		t.Fatalf("write with-project castfile: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "without-project"), 0o755); err != nil {
		t.Fatalf("mkdir without-project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "without-project", "notes.txt"), []byte("ignore\n"), 0o644); err != nil {
		t.Fatalf("write without-project notes: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	completions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, ".")
	values := completionValuesOnly(completions)

	if !slices.Contains(values, "with-project/") {
		t.Fatalf("expected with-project/ in completions, got %v", values)
	}
	if slices.Contains(values, "without-project/") {
		t.Fatalf("did not expect without-project/ in completions, got %v", values)
	}
}

func TestProvideProjectFlagCompletion_ShowsAliasesOnlyWithWorkspace(t *testing.T) {
	tmpDir := t.TempDir()

	rootFile := filepath.Join(tmpDir, "castfile")
	serviceDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("mkdir service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "castfile"), []byte("name: api\n"), 0o644); err != nil {
		t.Fatalf("write service castfile: %v", err)
	}

	withWorkspace := "name: root\nworkspace:\n  aliases:\n    api: services/api\n"
	if err := os.WriteFile(rootFile, []byte(withWorkspace), 0o644); err != nil {
		t.Fatalf("write root castfile with workspace: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	withAliasCompletions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, "@")
	if !slices.Contains(withAliasCompletions, "@api") {
		t.Fatalf("expected @api alias completion with workspace, got %v", withAliasCompletions)
	}

	withoutWorkspace := "name: root\n"
	if err := os.WriteFile(rootFile, []byte(withoutWorkspace), 0o644); err != nil {
		t.Fatalf("write root castfile without workspace: %v", err)
	}

	withoutAliasCompletions, _ := provideProjectFlagCompletion(&cobra.Command{}, nil, "@")
	if slices.Contains(withoutAliasCompletions, "@api") {
		t.Fatalf("did not expect @api alias completion without workspace, got %v", withoutAliasCompletions)
	}
}

func TestProvideContextFlagCompletion_ProjectFolderUsesLocalProjectFile(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "service")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	projectContent := "name: service\nconfig:\n  context: prod\n  contexts: [prod, qa]\n"
	if err := os.WriteFile(filepath.Join(projectDir, "castfile"), []byte(projectContent), 0o644); err != nil {
		t.Fatalf("write service castfile: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", "service", "")
	cmd.Flags().StringP("context", "c", "", "")

	completions, _ := provideContextFlagCompletion(cmd, []string{"--project", "service"}, "")
	for _, want := range []string{"default", "prod", "qa"} {
		if !slices.Contains(completions, want) {
			t.Fatalf("expected context completion %q from folder project, got %v", want, completions)
		}
	}
}

func TestProvideContextFlagCompletion_ProjectFolderFallsBackToNearest(t *testing.T) {
	tmpDir := t.TempDir()
	rootContent := "name: root\nconfig:\n  context: dev\n  contexts: [dev, stage]\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "castfile"), []byte(rootContent), 0o644); err != nil {
		t.Fatalf("write root castfile: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "empty-folder"), 0o755); err != nil {
		t.Fatalf("mkdir empty-folder: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("project", "p", "empty-folder", "")
	cmd.Flags().StringP("context", "c", "", "")

	completions, _ := provideContextFlagCompletion(cmd, []string{"--project", "empty-folder"}, "")
	for _, want := range []string{"default", "dev", "stage"} {
		if !slices.Contains(completions, want) {
			t.Fatalf("expected fallback context completion %q, got %v", want, completions)
		}
	}
}
