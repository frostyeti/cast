package projects

import (
	stdexec "os/exec"
	"strconv"
	"strings"
)

var gitLsRemote = func(repoURL string) (string, int, error) {
	cmd := stdexec.Command("git", "ls-remote", "--tags", repoURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), 1, err
	}
	return string(out), 0, nil
}

type gitResolveMode int

const (
	gitResolveBranch gitResolveMode = iota
	gitResolveDefault
	gitResolveCommit
)

// resolveGitVersion checks if a requested version like "v1" can be resolved
// to a specific git tag by querying the remote repository.
func resolveGitReference(repoURL, version, subPath string) (string, gitResolveMode) {
	if version == "" || isHeadRef(version) {
		return "HEAD", gitResolveDefault
	}

	if isGitCommitRef(version) {
		return version, gitResolveCommit
	}

	if version == "main" || version == "master" {
		return version, gitResolveBranch
	}

	if !looksLikeRemoteVersion(version) {
		return version, gitResolveBranch
	}

	stdout, code, err := gitLsRemote(repoURL)
	if err != nil || code != 0 {
		return version, gitResolveBranch
	}

	lines := strings.Split(stdout, "\n")
	tags := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		tagRef := parts[1]
		if !strings.HasPrefix(tagRef, "refs/tags/") {
			continue
		}
		tag := strings.TrimPrefix(tagRef, "refs/tags/")
		if strings.HasSuffix(tag, "^{}") {
			continue
		}
		tags = append(tags, tag)
	}

	chooseBest := func(candidates []string) string {
		if strings.Contains(version, "-") {
			for _, t := range candidates {
				if t == version || strings.HasSuffix(t, "/"+version) {
					return t
				}
			}
			return ""
		}

		if version == "HEAD" {
			return ""
		}

		requested := trimVersion(version)
		var bestTag string
		var bestMajor, bestMinor, bestPatch int
		found := false

		for _, t := range candidates {
			base := t
			if idx := strings.LastIndex(base, "/"); idx > -1 {
				if subPath == "" {
					continue
				}
				base = base[idx+1:]
			}

			baseVersion := trimVersion(base)
			if baseVersion != requested && !strings.HasPrefix(baseVersion, requested+".") {
				continue
			}

			major, minor, patch, ok := parseVersionParts(base)
			if !ok {
				continue
			}

			if !found || major > bestMajor || (major == bestMajor && minor > bestMinor) || (major == bestMajor && minor == bestMinor && patch > bestPatch) {
				found = true
				bestMajor, bestMinor, bestPatch = major, minor, patch
				bestTag = t
			}
		}

		return bestTag
	}

	if subPath != "" {
		prefix := strings.TrimSuffix(subPath, "/") + "/"
		prefixed := make([]string, 0)
		for _, t := range tags {
			if strings.HasPrefix(t, prefix) {
				prefixed = append(prefixed, t)
			}
		}
		if best := chooseBest(prefixed); best != "" {
			return best, gitResolveBranch
		}
	}

	if best := chooseBest(tags); best != "" {
		return best, gitResolveBranch
	}

	return version, gitResolveBranch
}

func resolveGitVersion(repoURL, version, subPath string) string {
	resolved, _ := resolveGitReference(repoURL, version, subPath)
	return resolved
}

func trimVersion(v string) string {
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if i := strings.IndexByte(v, '-'); i > -1 {
		v = v[:i]
	}
	return v
}

func parseVersionParts(v string) (int, int, int, bool) {
	v = trimVersion(v)
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return 0, 0, 0, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, false
		}
	}

	return major, minor, patch, true
}
