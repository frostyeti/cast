package actions

import (
	"os"
	"path/filepath"
	"runtime"
)

func FindNearestGitRepoFromPath(path string) (string, error) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(path) + "\\"
	}

	currentDir := path
	for currentDir != root && currentDir != "." && currentDir != "" {
		tryPath := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(tryPath); err == nil {
			return currentDir, nil
		}

		currentDir = filepath.Dir(currentDir)
	}

	return "", os.ErrNotExist
}

func FindNearestWorkspaceConfigFromPath(path string) (string, error) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(path) + "\\"
	}

	currentDir := path
	for currentDir != root && currentDir != "." && currentDir != "" {
		tryPath := filepath.Join(currentDir, ".cast", "workspace.yaml")
		if _, err := os.Stat(tryPath); err == nil {
			return tryPath, nil
		}

		currentDir = filepath.Dir(currentDir)
	}

	return "", os.ErrNotExist
}

func FindNearestCastfileFromPath(path string) (string, error) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(path) + "\\"
	}

	currentDir := path
	for currentDir != root && currentDir != "." && currentDir != "" {
		tryPath := filepath.Join(currentDir, ".castfile")
		if _, err := os.Stat(tryPath); err == nil {
			return tryPath, nil
		}

		currentDir = filepath.Dir(currentDir)
	}

	return "", os.ErrNotExist
}
