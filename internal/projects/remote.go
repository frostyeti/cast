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

type remoteGitTarget struct {
	repoURL    string
	version    string
	subPath    string
	cacheParts []string
}

func splitVersionAndSubPath(ref string) (string, string) {
	if ref == "" {
		return "", ""
	}

	idx := strings.IndexRune(ref, '/')
	if idx < 0 {
		return ref, ""
	}

	return ref[:idx], ref[idx+1:]
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

	if strings.HasPrefix(repoPart, "git@") || strings.HasPrefix(repoPart, "https://") || strings.HasPrefix(repoPart, "http://") || strings.HasPrefix(repoPart, "file://") {
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
				"cast",
				"spell",
				"cast.yaml",
				"spell.yaml",
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

	if strings.HasPrefix(uses, "git@") || strings.HasPrefix(uses, "github:") || strings.HasPrefix(uses, "gh:") || strings.HasPrefix(uses, "gitlab:") || strings.HasPrefix(uses, "gl:") || strings.HasPrefix(uses, "azdo:") || strings.HasPrefix(uses, "spell:") || strings.HasPrefix(uses, "task:") || strings.HasPrefix(uses, "file://") || strings.HasPrefix(uses, "https://") || strings.HasPrefix(uses, "http://") {
		target, err := parseRemoteGitTarget(uses)
		if err != nil {
			return "", err
		}

		if len(target.version) == 0 {
			return "", errors.Newf("invalid remote task identifier, version required: %s", uses)
		}

		resolvedVersion := resolveGitVersion(target.repoURL, target.version, target.subPath)
		if len(target.cacheParts) > 0 {
			parts := append([]string{cacheDir}, target.cacheParts...)
			parts = append(parts, resolvedVersion)
			taskDir = filepath.Join(parts...)
		}

		repoURL := target.repoURL

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

		entryFile := taskDir
		if target.subPath != "" {
			entryFile = filepath.Join(taskDir, target.subPath)
		}

		// Check if it's a directory, if so look for cast.task or standard entrypoints
		stat, err := os.Stat(entryFile)
		if err == nil && stat.IsDir() {
			tryFiles := []string{
				"cast", "cast.yaml", "spell", "spell.yaml",
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
