package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/projects"
)

func resolveWorkspaceProjectByAlias(project *projects.Project, projectName string) (*projects.ProjectInfo, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}

	if err := project.InitWorkspace(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(projectName)
	name = strings.TrimPrefix(name, "@")
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("workspace project name is empty")
	}

	if workspaceProject, ok := project.Workspace[name]; ok {
		return workspaceProject, nil
	}

	for alias, workspaceProject := range project.Workspace {
		if strings.EqualFold(alias, name) {
			return workspaceProject, nil
		}
	}

	namePath := name
	if filepath.IsAbs(namePath) {
		if resolved, ok := resolveWorkspacePathInEntries(project, namePath); ok {
			return resolved, nil
		}
	} else {
		try := []string{
			filepath.Join(project.Dir, namePath),
			filepath.Join(project.Dir, strings.TrimPrefix(namePath, "./")),
		}

		for _, candidate := range try {
			if resolved, ok := resolveWorkspacePathInEntries(project, candidate); ok {
				return resolved, nil
			}
		}
	}

	return nil, errors.Newf("project '%s' not found in workspace", name)
}

func resolveWorkspacePathInEntries(project *projects.Project, target string) (*projects.ProjectInfo, bool) {
	if project == nil {
		return nil, false
	}

	for _, workspaceProject := range project.WorkspaceEntries {
		if workspaceProject == nil {
			continue
		}

		entryPath := workspaceProject.Path
		entryDir := filepath.Dir(entryPath)

		if samePath(target, entryPath) || samePath(target, entryDir) || samePath(target, workspaceProject.Rel) {
			return workspaceProject, true
		}
	}

	for _, workspaceProject := range project.Workspace {
		if workspaceProject == nil {
			continue
		}
		entryPath := workspaceProject.Path
		entryDir := filepath.Dir(entryPath)
		if samePath(target, entryPath) || samePath(target, entryDir) || samePath(target, workspaceProject.Rel) {
			return workspaceProject, true
		}
	}

	return nil, false
}

func samePath(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}

	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)

	if cleanA == cleanB {
		return true
	}

	absA, errA := filepath.Abs(cleanA)
	absB, errB := filepath.Abs(cleanB)
	if errA == nil && errB == nil && absA == absB {
		return true
	}
	if errA == nil && errB == nil {
		if relA, err := filepath.Rel(absA, absB); err == nil {
			if relA == "." {
				return true
			}
		}
	}

	return false
}

func resolveProjectFileByFolder(folder string) (string, bool) {
	name := strings.TrimSpace(folder)
	if name == "" {
		return "", false
	}

	for _, candidate := range []string{"castfile", ".castfile", "castfile.yaml", "castfile.yml", "cast.yaml", "cast.yml"} {
		path := filepath.Join(name, candidate)
		if info, err := os.Stat(path); err == nil && info != nil && !info.IsDir() {
			return path, true
		}
	}

	return "", false
}
