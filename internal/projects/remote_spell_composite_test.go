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

func TestRunRemoteTaskFetchNoticeAndSpacing(t *testing.T) {
	tmpDir := t.TempDir()

	repoDir := filepath.Join(tmpDir, "remote-repo")
	spellDir := filepath.Join(repoDir, "examples", "hello-world")
	if err := os.MkdirAll(spellDir, 0o755); err != nil {
		t.Fatalf("mkdir remote repo: %v", err)
	}

	spellPath := filepath.Join(spellDir, "spell")
	spellYaml := "" +
		"name: examples/hello-world\n" +
		"description: runs hello world\n" +
		"runs:\n" +
		"  using: composite\n" +
		"  steps:\n" +
		"    - uses: bash\n" +
		"      run: echo hello fetched\n"
	if err := os.WriteFile(spellPath, []byte(spellYaml), 0o644); err != nil {
		t.Fatalf("write spell: %v", err)
	}

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "initial")
	runGit(t, repoDir, "tag", "v1.0.0")

	projectDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}

	projectFile := filepath.Join(projectDir, "castfile")
	if err := os.WriteFile(projectFile, []byte("name: fetch-notice-test\n"), 0o644); err != nil {
		t.Fatalf("write project file: %v", err)
	}

	uses := "file://" + filepath.ToSlash(repoDir) + "@v1.0.0/examples/hello-world"
	newContext := func(stdout, stderr *bytes.Buffer) TaskContext {
		return TaskContext{
			Project: &Project{
				Dir:     projectDir,
				File:    projectFile,
				CastDir: filepath.Join(projectDir, ".cast"),
				Schema: types.Project{
					TrustedSources: []string{"file://"},
				},
			},
			Task: &Task{
				Id:   "fetch-hello",
				Uses: uses,
				Env:  map[string]string{},
				With: map[string]any{},
			},
			Context: context.Background(),
			Stdout:  stdout,
			Stderr:  stderr,
		}
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	res := runRemoteTask(newContext(stdout, stderr))
	if res.Status != runstatus.Ok || res.Err != nil {
		t.Fatalf("expected remote task to succeed, got status=%s err=%v stderr=%s", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	notice := "Fetching task: " + uses
	output := stdout.String()
	if !strings.Contains(output, notice+"\n\n") {
		t.Fatalf("expected fetch notice and separating blank line, got: %q", output)
	}
	if strings.Contains(output, "Cloning into") {
		t.Fatalf("expected git stdout to stay hidden without debug logging, got: %q", output)
	}
	if !strings.Contains(output, "hello fetched") {
		t.Fatalf("expected task output after fetch notice, got: %q", output)
	}

	stdout.Reset()
	stderr.Reset()
	res = runRemoteTask(newContext(stdout, stderr))
	if res.Status != runstatus.Ok || res.Err != nil {
		t.Fatalf("expected cached remote task to succeed, got status=%s err=%v stderr=%s", runstatus.ToString(res.Status), res.Err, stderr.String())
	}

	second := stdout.String()
	if strings.Contains(second, notice) {
		t.Fatalf("expected no fetch notice when task is cached, got: %q", second)
	}
	if !strings.Contains(second, "hello fetched") {
		t.Fatalf("expected cached task output, got: %q", second)
	}

	matches, err := filepath.Glob(filepath.Join(projectDir, ".cast", "cache", "tasks", "*", "examples-hello-world", "v1.0.0", "repo"))
	if err != nil {
		t.Fatalf("expected sparse repo cache glob to succeed, got error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("expected sparse repo cache dir to exist for file:// remote task")
	}

	cacheRepoDir := matches[0]
	entryDir := filepath.Join(cacheRepoDir, "examples", "hello-world")
	if _, err := os.Stat(entryDir); err != nil {
		t.Fatalf("expected sparse entry dir to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheRepoDir, "cast.task")); err == nil {
		t.Fatalf("did not expect root cast.task in sparse checkout for subpath")
	}
}
