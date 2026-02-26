package projects

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/go/exec"
)

// IsRemoteTask checks if the uses string indicates a remote task (e.g., github.com/..., @scope/...)
func IsRemoteTask(uses string) bool {
	return strings.HasPrefix(uses, "github.com/") ||
		strings.HasPrefix(uses, "https://") ||
		strings.HasPrefix(uses, "http://") ||
		strings.HasPrefix(uses, "@") ||
		strings.HasPrefix(uses, "jsr:") ||
		strings.HasPrefix(uses, "npm:") ||
		strings.HasPrefix(uses, "file://") ||
		strings.HasPrefix(uses, "./") ||
		strings.HasPrefix(uses, "../") ||
		filepath.IsAbs(uses)
}

// FetchRemoteTask resolves and downloads a remote task, returning the local file path to the entrypoint module.
func FetchRemoteTask(p *Project, uses string, trustedSources []string) (string, error) {
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
				"cast.yaml",
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

	cacheDir := filepath.Join(p.Dir, ".cast", "tasks")
	os.MkdirAll(cacheDir, 0755)

	hash := sha256.Sum256([]byte(uses))
	hashStr := hex.EncodeToString(hash[:])
	taskDir := filepath.Join(cacheDir, hashStr)

	// If it's a JSR or NPM package, Deno handles it natively via import "jsr:..." or "npm:..."
	// We don't necessarily need to download it manually if Deno wrapper will do it, but the prompt says:
	// "Download dependencies to a central cache directory (e.g., .cast/tasks/)."
	// "Git Tasks: Perform a shallow git clone or download a tarball for the specified tag/version."

	if strings.HasPrefix(uses, "github.com/") || strings.HasPrefix(uses, "file://") {
		// e.g., github.com/user/repo@v1.0.0
		// or file:///path/to/repo@v1.0.0
		parts := strings.Split(uses, "@")
		repoPath := parts[0]
		version := "main"
		if len(parts) > 1 {
			version = parts[1]
		}

		var repoURL string
		var subPath string

		if strings.HasPrefix(uses, "github.com/") {
			repoParts := strings.Split(repoPath, "/")
			if len(repoParts) < 3 {
				return "", errors.New("invalid github URI")
			}
			repoURL = fmt.Sprintf("https://%s/%s/%s.git", repoParts[0], repoParts[1], repoParts[2])
			if len(repoParts) > 3 {
				subPath = strings.Join(repoParts[3:], "/")
			}
		} else {
			// file:// protocol handling for local git repos
			repoURL = strings.TrimPrefix(repoPath, "file://")
			// It's just a local directory, subPath would be anything after the git root, but we can assume no subpath for basic local testing for now
			// Or we could try to resolve it properly.
			// Let's keep it simple for testing: assume no subPath for file://
		}

		resolvedVersion := resolveGitVersion(repoURL, version)

		if _, err := os.Stat(taskDir); os.IsNotExist(err) {
			cmd := exec.New("git", "clone", "--depth", "1", "--branch", resolvedVersion, repoURL, taskDir)
			out, err := cmd.Run()
			if err != nil || out.Code != 0 {
				// If branch failed, try cloning without branch (some repositories might not have standard main/master)
				cmd = exec.New("git", "clone", "--depth", "1", repoURL, taskDir)
				out, err = cmd.Run()
				if err != nil || out.Code != 0 {
					return "", errors.Newf("failed to clone remote task: %v\n%s", err, out.Stdout)
				}
			}
		}

		entryFile := filepath.Join(taskDir, subPath)
		// Check if it's a directory, if so look for cast.task or standard entrypoints
		stat, err := os.Stat(entryFile)
		if err == nil && stat.IsDir() {
			if _, err := os.Stat(filepath.Join(entryFile, "cast.task")); err == nil {
				return filepath.Join(entryFile, "cast.task"), nil
			}

			entrypoints := []string{
				"mod.ts", "main.ts", "index.ts",
				"mod.js", "main.js", "index.js",
			}
			for _, ep := range entrypoints {
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
