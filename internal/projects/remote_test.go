package projects

import (
	"bytes"
	stdErrors "errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemoteGitTarget(t *testing.T) {
	tests := []struct {
		name    string
		uses    string
		repoURL string
		version string
		subPath string
	}{
		{name: "github prefix", uses: "gh:org/repo@v1.2.3", repoURL: "https://github.com/org/repo.git", version: "v1.2.3", subPath: ""},
		{name: "gitlab prefix", uses: "gl:group/repo@v1.2.3/path/task", repoURL: "https://gitlab.com/group/repo.git", version: "v1.2.3", subPath: "path/task"},
		{name: "azure prefix", uses: "azdo:org/project/repo@v1.2.3/sub", repoURL: "https://dev.azure.com/org/project/_git/repo.git", version: "v1.2.3", subPath: "sub"},
		{name: "cast prefix", uses: "cast:spell@v1.0.0/tasks/hello", repoURL: "https://github.com/frostyeti/spells.git", version: "v1.0.0", subPath: "spell/tasks/hello"},
		{name: "ssh url", uses: "git@github.com:org/repo.git@v1.2.3/path", repoURL: "git@github.com:org/repo.git", version: "v1.2.3", subPath: "path"},
		{name: "https url", uses: "https://example.com/org/repo.git@main", repoURL: "https://example.com/org/repo.git", version: "main", subPath: ""},
		{name: "github.com ref", uses: "github.com/org/repo@head", repoURL: "https://github.com/org/repo.git", version: "head", subPath: ""},
		{name: "gitlab.com ref", uses: "gitlab.com/group/repo@abc1234", repoURL: "https://gitlab.com/group/repo.git", version: "abc1234", subPath: ""},
		{name: "azure devops ref", uses: "dev.azure.com/org/project/_git/repo@feature", repoURL: "https://dev.azure.com/org/project/_git/repo.git", version: "feature", subPath: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRemoteGitTarget(tt.uses)
			if err != nil {
				t.Fatalf("parseRemoteGitTarget() error = %v", err)
			}
			if got.repoURL != tt.repoURL {
				t.Fatalf("repoURL = %q, want %q", got.repoURL, tt.repoURL)
			}
			if got.version != tt.version {
				t.Fatalf("version = %q, want %q", got.version, tt.version)
			}
			if got.subPath != tt.subPath {
				t.Fatalf("subPath = %q, want %q", got.subPath, tt.subPath)
			}
		})
	}
}

func TestIsRemoteTask(t *testing.T) {
	for _, uses := range []string{"gh:org/repo@v1", "gl:group/repo@v1", "azdo:org/project/repo@v1", "cast:spell@v1", "git@github.com:org/repo.git@v1"} {
		if !IsRemoteTask(uses) {
			t.Fatalf("expected %q to be remote", uses)
		}
	}
	for _, uses := range []string{"github.com/org/repo@head", "gitlab.com/group/repo@abc1234", "dev.azure.com/org/project/_git/repo@main", "ssh://git.example.com/repo.git@head"} {
		if !IsRemoteTask(uses) {
			t.Fatalf("expected %q to be remote", uses)
		}
	}
}

func TestRemoteVersionHelpers(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		head bool
		sha  bool
	}{
		{name: "head", ref: "head", head: true},
		{name: "HEAD", ref: "HEAD", head: true},
		{name: "sha", ref: "abc1234", sha: true},
		{name: "tag", ref: "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHeadRef(tt.ref); got != tt.head {
				t.Fatalf("isHeadRef(%q) = %v, want %v", tt.ref, got, tt.head)
			}
			if got := isGitCommitRef(tt.ref); got != tt.sha {
				t.Fatalf("isGitCommitRef(%q) = %v, want %v", tt.ref, got, tt.sha)
			}
		})
	}
}

func TestResolveGitVersion_PrereleaseExact(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repoDir, "cast.task"), []byte("name: test\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "initial")
	runGit(t, repoDir, "tag", "v1.2.3")
	runGit(t, repoDir, "tag", "task/v1.2.3")
	runGit(t, repoDir, "tag", "v2.3.1-beta.1")

	if got := resolveGitVersion(repoDir, "v1", ""); got != "v1.2.3" {
		t.Fatalf("resolveGitVersion(v1) = %q, want v1.2.3", got)
	}

	if got := resolveGitVersion(repoDir, "v1.2.3", "task"); got != "task/v1.2.3" {
		t.Fatalf("resolveGitVersion(subpath) = %q, want task/v1.2.3", got)
	}

	if got := resolveGitVersion(repoDir, "v2.3.1-beta.1", ""); got != "v2.3.1-beta.1" {
		t.Fatalf("resolveGitVersion(prerelease) = %q, want exact match", got)
	}
}

func TestResolveGitReference_HeadBranchAndSha(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(repoDir, "cast.task"), []byte("name: test\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "initial")
	runGit(t, repoDir, "checkout", "-b", "feature/login")
	sha := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "HEAD"))

	if got, mode := resolveGitReference(repoDir, "head", ""); got != "HEAD" || mode != gitResolveDefault {
		t.Fatalf("resolveGitReference(head) = (%q, %v), want (HEAD, default)", got, mode)
	}

	if got, mode := resolveGitReference(repoDir, sha, ""); got != sha || mode != gitResolveCommit {
		t.Fatalf("resolveGitReference(sha) = (%q, %v), want (%q, commit)", got, mode, sha)
	}

	if got, mode := resolveGitReference(repoDir, "feature/login", ""); got != "feature/login" || mode != gitResolveBranch {
		t.Fatalf("resolveGitReference(branch) = (%q, %v), want (feature/login, branch)", got, mode)
	}
}

func TestWriteGitStdoutIfDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	payload := []byte("git stdout line")

	t.Setenv("CAST_DEBUG", "0")
	writeGitStdoutIfDebug(buf, payload)
	if got := buf.String(); got != "" {
		t.Fatalf("expected no git stdout without debug logging, got %q", got)
	}

	t.Setenv("CAST_DEBUG", "1")
	writeGitStdoutIfDebug(buf, payload)
	if got := buf.String(); got != "git stdout line\n" {
		t.Fatalf("expected git stdout with trailing newline in debug mode, got %q", got)
	}
}

func TestFormatGitCommandError_DebugGate(t *testing.T) {
	gErr := stdErrors.New("boom")

	t.Setenv("CAST_DEBUG", "0")
	err := formatGitCommandError("failed to clone remote task", gErr, 1, []byte("sensitive clone details\n"))
	if strings.Contains(err.Error(), "sensitive clone details") {
		t.Fatalf("expected git stdout to be hidden without debug logging, got %q", err.Error())
	}

	t.Setenv("CAST_DEBUG", "1")
	err = formatGitCommandError("failed to clone remote task", gErr, 1, []byte("debug clone details\n"))
	if !strings.Contains(err.Error(), "debug clone details") {
		t.Fatalf("expected git stdout to be shown with debug logging, got %q", err.Error())
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return string(out)
}
