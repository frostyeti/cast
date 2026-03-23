package projects

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/go/exec"
)

const (
	CastRemoteTasksDirEnv         = "CAST_REMOTE_TASKS_DIR"
	CastVolatileRemoteTasksDirEnv = "CAST_VOLATILE_REMOTE_TASKS_DIR"
)

type FetchRemoteTaskOptions struct {
	Stdout       io.Writer
	ForceRefresh bool
}

type remoteGitTarget struct {
	repoURL    string
	version    string
	subPath    string
	cacheParts []string
}

type remoteCacheLayout struct {
	repoDir  string
	entryDir string
	subPath  string
}

func resolveConfiguredRemoteTasksDir(projectDir, envKey, defaultPath string) string {
	dir := strings.TrimSpace(os.Getenv(envKey))
	if dir == "" {
		dir = defaultPath
	}

	if filepath.IsAbs(dir) {
		return dir
	}

	return filepath.Join(projectDir, dir)
}

func stableRemoteTasksDir(projectDir string) string {
	homeDir, _ := os.UserHomeDir()
	defaultDir := filepath.Join(homeDir, ".local", "cast", "tasks")
	return resolveConfiguredRemoteTasksDir(projectDir, CastRemoteTasksDirEnv, defaultDir)
}

func volatileRemoteTasksDir(p *Project) string {
	defaultDir := filepath.Join(p.Dir, ".cast", "cache", "tasks")
	return resolveConfiguredRemoteTasksDir(p.Dir, CastVolatileRemoteTasksDirEnv, defaultDir)
}

// ResolveRemoteTasksDir resolves the stable remote tasks directory.
func ResolveRemoteTasksDir(projectDir string) string {
	return stableRemoteTasksDir(projectDir)
}

// ResolveVolatileRemoteTasksDir resolves the volatile remote tasks directory.
func ResolveVolatileRemoteTasksDir(projectDir string) string {
	defaultDir := filepath.Join(projectDir, ".cast", "cache", "tasks")
	return resolveConfiguredRemoteTasksDir(projectDir, CastVolatileRemoteTasksDirEnv, defaultDir)
}

func isVolatileRemoteReference(version string, mode gitResolveMode) bool {
	if version == "" || isHeadRef(version) {
		return true
	}

	if mode == gitResolveCommit {
		return false
	}

	if version == "main" || version == "master" {
		return true
	}

	if !looksLikeRemoteVersion(version) {
		return true
	}

	return false
}

// IsVolatileRemoteTaskRef reports if a remote reference is branch/head based.
func IsVolatileRemoteTaskRef(uses string) bool {
	if !IsRemoteTask(uses) {
		return false
	}

	target, err := parseRemoteGitTarget(uses)
	if err != nil {
		return false
	}

	_, mode := resolveGitReference(target.repoURL, target.version, target.subPath)
	return isVolatileRemoteReference(target.version, mode)
}

func remoteTaskStdoutWriter(stdout io.Writer) io.Writer {
	if stdout != nil {
		return stdout
	}

	return os.Stdout
}

func isDebugLoggingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("CAST_DEBUG")))
	switch v {
	case "1", "true", "yes", "on", "debug":
		return true
	default:
		return false
	}
}

func writeGitStdoutIfDebug(stdout io.Writer, gitStdout []byte) {
	if !isDebugLoggingEnabled() || len(gitStdout) == 0 {
		return
	}

	writer := remoteTaskStdoutWriter(stdout)
	_, _ = writer.Write(gitStdout)
	if !bytes.HasSuffix(gitStdout, []byte("\n")) {
		_, _ = io.WriteString(writer, "\n")
	}
}

func formatGitCommandError(prefix string, err error, code int, gitStdout []byte) error {
	if err == nil {
		err = errors.Newf("exit code %d", code)
	}

	if isDebugLoggingEnabled() && len(bytes.TrimSpace(gitStdout)) > 0 {
		return errors.Newf("%s: %v\n%s", prefix, err, string(gitStdout))
	}

	return errors.Newf("%s: %v", prefix, err)
}

func runGitCommand(cmd *exec.Cmd) ([]byte, int, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.WithStdout(&stdout)
	cmd.WithStderr(&stderr)

	out, err := cmd.Run()
	code := 0
	if out != nil {
		code = out.Code
		if stdout.Len() == 0 && len(out.Stdout) > 0 {
			stdout.Write(out.Stdout)
		}
	}

	return stdout.Bytes(), code, err
}

func runGitOrError(cmd *exec.Cmd, stdout io.Writer, prefix string) error {
	gitStdout, code, err := runGitCommand(cmd)
	writeGitStdoutIfDebug(stdout, gitStdout)
	if err != nil || code != 0 {
		return formatGitCommandError(prefix, err, code, gitStdout)
	}

	return nil
}

func normalizeRemoteSubPath(subPath string) (string, error) {
	subPath = strings.TrimSpace(subPath)
	if subPath == "" {
		return "", nil
	}

	raw := strings.ReplaceAll(subPath, "\\", "/")
	for _, part := range strings.Split(raw, "/") {
		if part == ".." {
			return "", errors.Newf("invalid remote task subpath: %s", subPath)
		}
	}

	normalized := filepath.ToSlash(filepath.Clean("/" + subPath))
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" || normalized == "." {
		return "", nil
	}

	if normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", errors.Newf("invalid remote task subpath: %s", subPath)
	}

	return normalized, nil
}

func hyphenateRemoteSubPath(subPath string) string {
	slug := strings.ReplaceAll(subPath, "/", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "root"
	}

	return slug
}

func buildRemoteTaskCacheLayout(cacheDir, hashStr string, target remoteGitTarget, resolvedVersion string) (remoteCacheLayout, error) {
	layout := remoteCacheLayout{}

	normalizedSubPath, err := normalizeRemoteSubPath(target.subPath)
	if err != nil {
		return layout, err
	}

	layout.subPath = normalizedSubPath

	if len(target.cacheParts) > 0 {
		parts := append([]string{cacheDir}, target.cacheParts...)
		if normalizedSubPath != "" {
			slug := hyphenateRemoteSubPath(normalizedSubPath)
			parts = append(parts, slug, resolvedVersion, "repo")
		} else {
			parts = append(parts, resolvedVersion)
		}
		layout.repoDir = filepath.Join(parts...)
	} else {
		parts := []string{cacheDir, hashStr}
		if normalizedSubPath != "" {
			slug := hyphenateRemoteSubPath(normalizedSubPath)
			parts = append(parts, slug, resolvedVersion, "repo")
		}
		layout.repoDir = filepath.Join(parts...)
	}

	layout.entryDir = layout.repoDir
	if normalizedSubPath != "" {
		layout.entryDir = filepath.Join(layout.repoDir, filepath.FromSlash(normalizedSubPath))
	}

	return layout, nil
}

func cloneRemoteTaskRepository(repoURL, resolvedVersion string, cloneMode gitResolveMode, taskDir string, stdout io.Writer) error {
	var cloneCmd *exec.Cmd
	switch cloneMode {
	case gitResolveCommit:
		cloneCmd = exec.New("git", "clone", repoURL, taskDir)
	case gitResolveDefault:
		cloneCmd = exec.New("git", "clone", "--depth", "1", repoURL, taskDir)
	default:
		cloneCmd = exec.New("git", "clone", "--depth", "1", "--branch", resolvedVersion, repoURL, taskDir)
	}

	err := runGitOrError(cloneCmd, stdout, "failed to clone remote task")
	if err != nil {
		if cloneMode != gitResolveBranch {
			return err
		}

		_ = os.RemoveAll(taskDir)
		fallback := exec.New("git", "clone", "--depth", "1", repoURL, taskDir)
		if fallbackErr := runGitOrError(fallback, stdout, "failed to clone remote task"); fallbackErr != nil {
			return fallbackErr
		}
	}

	if cloneMode == gitResolveCommit {
		checkout := exec.New("git", "-C", taskDir, "checkout", "--detach", resolvedVersion)
		return runGitOrError(checkout, stdout, fmt.Sprintf("failed to checkout remote task commit %s", resolvedVersion))
	}

	return nil
}

func cloneRemoteTaskRepositorySparse(repoURL, resolvedVersion string, cloneMode gitResolveMode, taskDir, subPath string, stdout io.Writer) error {
	var cloneCmd *exec.Cmd
	switch cloneMode {
	case gitResolveCommit:
		cloneCmd = exec.New("git", "clone", "--filter=blob:none", "--no-checkout", repoURL, taskDir)
	case gitResolveDefault:
		cloneCmd = exec.New("git", "clone", "--filter=blob:none", "--depth", "1", "--no-checkout", repoURL, taskDir)
	default:
		cloneCmd = exec.New("git", "clone", "--filter=blob:none", "--depth", "1", "--branch", resolvedVersion, "--no-checkout", repoURL, taskDir)
	}

	err := runGitOrError(cloneCmd, stdout, "failed to clone remote task")
	if err != nil {
		if cloneMode != gitResolveBranch {
			return err
		}

		_ = os.RemoveAll(taskDir)
		fallback := exec.New("git", "clone", "--filter=blob:none", "--depth", "1", "--no-checkout", repoURL, taskDir)
		if fallbackErr := runGitOrError(fallback, stdout, "failed to clone remote task"); fallbackErr != nil {
			return fallbackErr
		}
	}

	if err := runGitOrError(exec.New("git", "-C", taskDir, "sparse-checkout", "init", "--no-cone"), stdout, "failed to initialize sparse checkout"); err != nil {
		return err
	}

	patterns := []string{"-C", taskDir, "sparse-checkout", "set", "--", subPath}
	if !strings.HasSuffix(subPath, "/") {
		patterns = append(patterns, subPath+"/**")
	}
	if err := runGitOrError(exec.New("git", patterns...), stdout, "failed to set sparse checkout path"); err != nil {
		return err
	}

	if cloneMode == gitResolveCommit {
		checkout := exec.New("git", "-C", taskDir, "checkout", "--detach", resolvedVersion)
		return runGitOrError(checkout, stdout, fmt.Sprintf("failed to checkout remote task commit %s", resolvedVersion))
	}

	return runGitOrError(exec.New("git", "-C", taskDir, "checkout"), stdout, "failed to checkout sparse remote task")
}

func splitVersionAndSubPath(ref string) (string, string) {
	if ref == "" {
		return "", ""
	}

	idx := strings.IndexRune(ref, '/')
	if idx < 0 {
		return ref, ""
	}

	if !looksLikeRemoteVersion(ref[:idx]) {
		return ref, ""
	}

	return ref[:idx], ref[idx+1:]
}

func isHeadRef(ref string) bool {
	return strings.EqualFold(strings.TrimSpace(ref), "head")
}

func isGitCommitRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}

	for _, r := range ref {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}

	return true
}

func looksLikeRemoteVersion(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}

	if isHeadRef(ref) || isGitCommitRef(ref) {
		return true
	}

	base := ref
	if idx := strings.IndexByte(base, '-'); idx > -1 {
		base = base[:idx]
	}

	base = strings.TrimPrefix(strings.TrimPrefix(base, "v"), "V")
	parts := strings.Split(base, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	return true
}

func joinSubPath(base, extra string) string {
	base = strings.TrimSuffix(base, "/")
	extra = strings.TrimPrefix(extra, "/")
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + "/" + extra
}

func trimGitSuffix(v string) string {
	return strings.TrimSuffix(v, ".git")
}

func parseHostedRemoteTarget(repoPart string) (string, []string, bool) {
	switch {
	case strings.HasPrefix(repoPart, "github.com/"):
		path := strings.TrimPrefix(repoPart, "github.com/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return "", nil, false
		}
		repo := trimGitSuffix(parts[1])
		return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], repo), []string{"github", parts[0], repo}, true
	case strings.HasPrefix(repoPart, "gitlab.com/"):
		path := strings.TrimPrefix(repoPart, "gitlab.com/")
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return "", nil, false
		}
		repo := trimGitSuffix(parts[1])
		return fmt.Sprintf("https://gitlab.com/%s/%s.git", parts[0], repo), []string{"gitlab", parts[0], repo}, true
	case strings.HasPrefix(repoPart, "dev.azure.com/"):
		path := strings.TrimPrefix(repoPart, "dev.azure.com/")
		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			return "", nil, false
		}

		org := parts[0]
		project := parts[1]
		repo := ""
		switch {
		case len(parts) == 3:
			repo = parts[2]
		case len(parts) == 4 && parts[2] == "_git":
			repo = parts[3]
		default:
			return "", nil, false
		}

		repo = trimGitSuffix(repo)
		return fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s.git", org, project, repo), []string{"azdo", org, project, repo}, true
	default:
		return "", nil, false
	}
}

func parseRemoteGitTarget(uses string) (remoteGitTarget, error) {
	result := remoteGitTarget{version: "v1.0.0"}

	var repoPart, refPart string
	if idx := strings.LastIndex(uses, "@"); idx > -1 {
		repoPart = uses[:idx]
		refPart = uses[idx+1:]
	} else {
		repoPart = uses
	}

	if v, sub := splitVersionAndSubPath(refPart); v != "" {
		result.version = v
		result.subPath = sub
	}

	if repoURL, cacheParts, ok := parseHostedRemoteTarget(repoPart); ok {
		result.repoURL = repoURL
		result.cacheParts = cacheParts
		return result, nil
	}

	if strings.HasPrefix(repoPart, "git@") || strings.HasPrefix(repoPart, "ssh://") || strings.HasPrefix(repoPart, "git+ssh://") || strings.HasPrefix(repoPart, "https://") || strings.HasPrefix(repoPart, "http://") || strings.HasPrefix(repoPart, "file://") {
		result.repoURL = repoPart
		if strings.HasPrefix(repoPart, "git@") && refPart == "" {
			return result, errors.Newf("invalid remote task identifier, version required: %s", uses)
		}
		return result, nil
	}

	prefix, path, ok := strings.Cut(repoPart, ":")
	if !ok {
		return result, errors.Newf("invalid remote task identifier: %s", uses)
	}

	switch prefix {
	case "gh", "github":
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return result, errors.Newf("invalid remote task identifier for github: %s", uses)
		}
		result.repoURL = fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1])
		result.cacheParts = []string{"github", parts[0], parts[1]}
	case "gl", "gitlab":
		parts := strings.Split(path, "/")
		if len(parts) != 2 {
			return result, errors.Newf("invalid remote task identifier for gitlab: %s", uses)
		}
		result.repoURL = fmt.Sprintf("https://gitlab.com/%s/%s.git", parts[0], parts[1])
		result.cacheParts = []string{"gitlab", parts[0], parts[1]}
	case "azdo":
		parts := strings.Split(path, "/")
		if len(parts) < 2 || len(parts) > 3 {
			return result, errors.Newf("invalid remote task identifier for azdo: %s", uses)
		}
		repo := parts[1]
		if len(parts) == 3 {
			repo = parts[2]
		}
		result.repoURL = fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s.git", parts[0], parts[1], repo)
		result.cacheParts = []string{"azdo", parts[0], parts[1], repo}
	case "spell", "task", "cast":
		result.repoURL = "https://github.com/frostyeti/spells.git"
		result.subPath = joinSubPath(path, result.subPath)
		result.cacheParts = []string{"cast"}
	default:
		return result, errors.Newf("unsupported remote task URI: %s", uses)
	}

	return result, nil
}

// IsRemoteTask checks if the uses string indicates a remote task (e.g., github.com/..., @scope/...)
func IsRemoteTask(uses string) bool {
	return strings.HasPrefix(uses, "spell:") ||
		strings.HasPrefix(uses, "task:") ||
		strings.HasPrefix(uses, "cast:") ||
		strings.HasPrefix(uses, "github:") ||
		strings.HasPrefix(uses, "gh:") ||
		strings.HasPrefix(uses, "gl:") ||
		strings.HasPrefix(uses, "gitlab:") ||
		strings.HasPrefix(uses, "azdo:") ||
		strings.HasPrefix(uses, "git@") ||
		strings.HasPrefix(uses, "ssh://") ||
		strings.HasPrefix(uses, "git+ssh://") ||
		strings.HasPrefix(uses, "https://") ||
		strings.HasPrefix(uses, "http://") ||
		strings.HasPrefix(uses, "github.com/") ||
		strings.HasPrefix(uses, "gitlab.com/") ||
		strings.HasPrefix(uses, "dev.azure.com/") ||
		strings.HasPrefix(uses, "jsr:") ||
		strings.HasPrefix(uses, "npm:") ||
		strings.HasPrefix(uses, "file://") ||
		strings.HasPrefix(uses, "./") ||
		strings.HasPrefix(uses, "../") ||
		filepath.IsAbs(uses)
}

// FetchRemoteTask resolves and downloads a remote task, returning the local file path to the entrypoint module.
func fetchRemoteTaskWithOptions(p *Project, uses string, trustedSources []string, opts FetchRemoteTaskOptions) (string, error) {
	stdoutWriter := remoteTaskStdoutWriter(opts.Stdout)

	// First, check trusted sources
	isTrusted := false
	if len(trustedSources) == 0 || strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "../") || filepath.IsAbs(uses) {
		isTrusted = true
	}

	for _, pattern := range trustedSources {
		match, _ := filepath.Match(pattern, uses)
		if match || strings.HasPrefix(uses, pattern) {
			isTrusted = true
			break
		}
	}

	if len(trustedSources) > 0 && !isTrusted {
		return "", errors.Newf("remote task '%s' is not in trusted_sources", uses)
	}

	if strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "../") || filepath.IsAbs(uses) {
		entryFile := uses
		if !filepath.IsAbs(entryFile) {
			entryFile = filepath.Join(p.Dir, entryFile)
		}

		stat, err := os.Stat(entryFile)
		if err == nil && stat.IsDir() {
			possibleFiles := []string{
				"cast.task",
				"cast",
				"spell",
				"cast.yaml",
				"cast.yml",
				"spell.yaml",
				"spell.yml",
				"mod.ts", "main.ts", "index.ts",
				"mod.js", "main.js", "index.js",
			}
			for _, file := range possibleFiles {
				if _, err := os.Stat(filepath.Join(entryFile, file)); err == nil {
					return filepath.Join(entryFile, file), nil
				}
			}
		}
		return entryFile, nil
	}

	VerifyChecksumAndRefresh(p)

	hash := sha256.Sum256([]byte(uses))
	hashStr := hex.EncodeToString(hash[:])
	taskDir := ""

	// If it's a JSR or NPM package, Deno handles it natively via import "jsr:..." or "npm:..."
	// We don't necessarily need to download it manually if Deno wrapper will do it, but the prompt says:
	// "Download dependencies to a central cache directory (e.g., .cast/cache/tasks/)."
	// "Git Tasks: Perform a shallow git clone or download a tarball for the specified tag/version."

	if strings.HasPrefix(uses, "git@") || strings.HasPrefix(uses, "ssh://") || strings.HasPrefix(uses, "git+ssh://") || strings.HasPrefix(uses, "github:") || strings.HasPrefix(uses, "gh:") || strings.HasPrefix(uses, "gitlab:") || strings.HasPrefix(uses, "gl:") || strings.HasPrefix(uses, "azdo:") || strings.HasPrefix(uses, "spell:") || strings.HasPrefix(uses, "task:") || strings.HasPrefix(uses, "file://") || strings.HasPrefix(uses, "https://") || strings.HasPrefix(uses, "http://") || strings.HasPrefix(uses, "github.com/") || strings.HasPrefix(uses, "gitlab.com/") || strings.HasPrefix(uses, "dev.azure.com/") {
		target, err := parseRemoteGitTarget(uses)
		if err != nil {
			return "", err
		}

		if len(target.version) == 0 {
			return "", errors.Newf("invalid remote task identifier, version required: %s", uses)
		}

		normalizedSubPath, err := normalizeRemoteSubPath(target.subPath)
		if err != nil {
			return "", err
		}
		target.subPath = normalizedSubPath

		resolvedVersion, cloneMode := resolveGitReference(target.repoURL, target.version, target.subPath)
		cacheDir := volatileRemoteTasksDir(p)
		if !isVolatileRemoteReference(target.version, cloneMode) {
			cacheDir = stableRemoteTasksDir(p.Dir)
		}

		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return "", err
		}

		layout, err := buildRemoteTaskCacheLayout(cacheDir, hashStr, target, resolvedVersion)
		if err != nil {
			return "", err
		}
		taskDir = layout.repoDir

		repoURL := target.repoURL

		if opts.ForceRefresh {
			_ = os.RemoveAll(taskDir)
		}

		if _, err := os.Stat(taskDir); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(taskDir), 0o755); err != nil {
				return "", err
			}

			fmt.Fprintf(stdoutWriter, "Fetching task: %s\n", uses)

			if layout.subPath != "" {
				err = cloneRemoteTaskRepositorySparse(repoURL, resolvedVersion, cloneMode, taskDir, layout.subPath, stdoutWriter)
			} else {
				err = cloneRemoteTaskRepository(repoURL, resolvedVersion, cloneMode, taskDir, stdoutWriter)
			}
			if err != nil {
				return "", err
			}

			fmt.Fprintln(stdoutWriter)
		}

		entryFile := layout.entryDir

		// Check if it's a directory, if so look for cast.task or standard entrypoints
		stat, err := os.Stat(entryFile)
		if err == nil && stat.IsDir() {
			tryFiles := []string{
				"cast.task", "cast", "cast.yaml", "cast.yml", "spell", "spell.yaml", "spell.yml",
				"mod.ts", "main.ts", "index.ts",
				"mod.js", "main.js", "index.js",
			}
			for _, ep := range tryFiles {
				if _, err := os.Stat(filepath.Join(entryFile, ep)); err == nil {
					return filepath.Join(entryFile, ep), nil
				}
			}
		}
		return entryFile, nil
	} else if strings.HasPrefix(uses, "jsr:") || strings.HasPrefix(uses, "npm:") {
		// For JSR/NPM, we can just return the URI itself and let Deno's native module resolution handle it in the wrapper
		// Or we can cache it. "Fetch the manifest and module using standard HTTP requests or Deno's tooling."
		// Returning the string allows the Deno wrapper to just import it!
		return uses, nil
	}

	return "", errors.Newf("unsupported remote task URI: %s", uses)
}

// FetchRemoteTask resolves and downloads a remote task, returning the local file path to the entrypoint module.
func FetchRemoteTask(p *Project, uses string, trustedSources []string, stdout io.Writer) (string, error) {
	return fetchRemoteTaskWithOptions(p, uses, trustedSources, FetchRemoteTaskOptions{Stdout: stdout})
}

// FetchRemoteTaskWithOptions resolves and downloads a remote task with control options.
func FetchRemoteTaskWithOptions(p *Project, uses string, trustedSources []string, opts FetchRemoteTaskOptions) (string, error) {
	return fetchRemoteTaskWithOptions(p, uses, trustedSources, opts)
}
