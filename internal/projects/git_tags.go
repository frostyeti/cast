package projects

import (
	"strings"

	"github.com/Masterminds/semver"
	"github.com/frostyeti/go/exec"
)

// resolveGitVersion checks if a requested version like "v1" can be resolved
// to a specific semver tag like "v1.2.3" by querying the remote repository.
func resolveGitVersion(repoURL, version string) string {
	// If it looks like a branch name or 'main' / 'master', skip semver resolution
	if version == "" || version == "main" || version == "master" {
		return version
	}

	cmd := exec.New("git", "ls-remote", "--tags", repoURL)
	out, err := cmd.Run()
	if err != nil || out.Code != 0 {
		return version // Fallback to whatever was requested
	}

	lines := strings.Split(string(out.Stdout), "\n")
	var tags []string
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			tagRef := parts[1]
			// e.g. refs/tags/v1.0.0
			if strings.HasPrefix(tagRef, "refs/tags/") {
				tag := strings.TrimPrefix(tagRef, "refs/tags/")
				// Skip peeled tags ending in ^{}
				if !strings.HasSuffix(tag, "^{}") {
					tags = append(tags, tag)
				}
			}
		}
	}

	// Try to match the provided version as a constraint (e.g., "~1.x.x" if version is "v1")
	constraintStr := version
	if !strings.Contains(constraintStr, ".") {
		constraintStr = "^" + constraintStr // v1 becomes ^v1
	} else if strings.Count(constraintStr, ".") == 1 {
		constraintStr = "~" + constraintStr // v1.2 becomes ~v1.2
	}

	constraint, err := semver.NewConstraint(constraintStr)
	if err != nil {
		// Try fallback if the string is just "v1" directly matching a tag "v1"
		for _, t := range tags {
			if t == version {
				return version
			}
		}
		return version
	}

	var bestMatch *semver.Version
	bestTag := ""

	for _, t := range tags {
		v, err := semver.NewVersion(t)
		if err != nil {
			continue // skip invalid semver tags
		}

		if constraint.Check(v) {
			if bestMatch == nil || v.GreaterThan(bestMatch) {
				bestMatch = v
				bestTag = t
			}
		}
	}

	if bestTag != "" {
		return bestTag
	}

	return version // Fallback
}
