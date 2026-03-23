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

type remoteGitTarget struct {
	repoURL    string
	version    string
	subPath    string
	cacheParts []string
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
		strings.HasPrefix(uses, "@") ||
		strings.HasPrefix(uses, "jsr:") ||
		strings.HasPrefix(uses, "npm:") ||
		strings.HasPrefix(uses, "file://") ||
		strings.HasPrefix(uses, "./") ||
		strings.HasPrefix(uses, "../") ||
		filepath.IsAbs(uses)
}

// FetchRemoteTask resolves and downloads a remote task, returning the local file path to the entrypoint module.
func FetchRemoteTask(p *Project, uses string, trustedSources []string, stdout io.Writer) (string, error) {
	stdoutWriter := remoteTaskStdoutWriter(stdout)

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

	cacheDir := filepath.Join(p.Dir, ".cast", "cache", "tasks")
	os.MkdirAll(cacheDir, 0755)

	hash := sha256.Sum256([]byte(uses))
	hashStr := hex.EncodeToString(hash[:])
	taskDir := filepath.Join(cacheDir, hashStr)

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

		resolvedVersion, cloneMode := resolveGitReference(target.repoURL, target.version, target.subPath)
		if len(target.cacheParts) > 0 {
			parts := append([]string{cacheDir}, target.cacheParts...)
			parts = append(parts, resolvedVersion)
			taskDir = filepath.Join(parts...)
		}

		repoURL := target.repoURL

		if _, err := os.Stat(taskDir); os.IsNotExist(err) {
			fmt.Fprintf(stdoutWriter, "Fetching task: %s\n", uses)

			var cmd *exec.Cmd
			switch cloneMode {
			case gitResolveCommit:
				cmd = exec.New("git", "clone", repoURL, taskDir)
			case gitResolveDefault:
				cmd = exec.New("git", "clone", "--depth", "1", repoURL, taskDir)
			default:
				cmd = exec.New("git", "clone", "--depth", "1", "--branch", resolvedVersion, repoURL, taskDir)
			}
			gitStdout, code, err := runGitCommand(cmd)
			writeGitStdoutIfDebug(stdoutWriter, gitStdout)
			if err != nil || code != 0 {
				if cloneMode == gitResolveBranch {
					// If branch failed, try cloning without branch (some repositories might not have standard main/master)
					cmd = exec.New("git", "clone", "--depth", "1", repoURL, taskDir)
					gitStdout, code, err = runGitCommand(cmd)
					writeGitStdoutIfDebug(stdoutWriter, gitStdout)
					if err != nil || code != 0 {
						return "", formatGitCommandError("failed to clone remote task", err, code, gitStdout)
					}
				} else {
					return "", formatGitCommandError("failed to clone remote task", err, code, gitStdout)
				}
			}

			if cloneMode == gitResolveCommit {
				cmd = exec.New("git", "-C", taskDir, "checkout", "--detach", resolvedVersion)
				gitStdout, code, err = runGitCommand(cmd)
				writeGitStdoutIfDebug(stdoutWriter, gitStdout)
				if err != nil || code != 0 {
					return "", formatGitCommandError(fmt.Sprintf("failed to checkout remote task commit %s", resolvedVersion), err, code, gitStdout)
				}
			}

			fmt.Fprintln(stdoutWriter)
		}

		entryFile := taskDir
		if target.subPath != "" {
			entryFile = filepath.Join(taskDir, target.subPath)
		}

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
	} else if strings.HasPrefix(uses, "jsr:") || strings.HasPrefix(uses, "npm:") || strings.HasPrefix(uses, "@") {
		// For JSR/NPM, we can just return the URI itself and let Deno's native module resolution handle it in the wrapper
		// Or we can cache it. "Fetch the manifest and module using standard HTTP requests or Deno's tooling."
		// Returning the string allows the Deno wrapper to just import it!
		if strings.HasPrefix(uses, "@") {
			return "jsr:" + uses, nil // Default to JSR for @scope
		}
		return uses, nil
	}

	return "", errors.Newf("unsupported remote task URI: %s", uses)
}
