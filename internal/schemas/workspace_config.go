package schemas

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/frostyeti/cast/internal/paths"
	"github.com/gobwas/glob"
	"go.yaml.in/yaml/v4"
)

type WorkspaceConfig struct {
	File     string
	Dir      string
	Id       string
	Projects *ProjectDiscovery
	Defaults *WorkspaceDefaults
	Env      *Env
	Modules  []string
}

func NewWorkspaceConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		Projects: &ProjectDiscovery{
			Include: []string{},
			Exclude: []string{"**/*./**", "**/node_modules/**", "**/.git/**", "**/bin/**", "**/obj/**"},
			Cache:   []CastfileInfo{},
		},
		Defaults: &WorkspaceDefaults{
			Context: "default",
			Shell:   "shell",
			Remote:  false,
		},
		Env:     &Env{},
		Modules: []string{},
	}
}

func NewWorkspaceConfigFromPath(path string) (*WorkspaceConfig, error) {
	ws := NewWorkspaceConfig()
	if strings.HasSuffix(path, "workspace.yaml") {
		ws.File = path
		ws.Dir = filepath.Dir(path)
	} else if strings.HasSuffix(path, ".cast") {
		ws.Dir = path
		ws.File = filepath.Join(path, "workspace.yaml")
	} else {
		ws.Dir = filepath.Join(path, ".cast")
		ws.File = filepath.Join(ws.Dir, "workspace.yaml")
	}

	ws.Projects.Cache = []CastfileInfo{}
	ws.Projects.Include = []string{"**"}
	ws.Projects.AutoDiscover.Enabled = true
	ws.Projects.AutoDiscover.Expires = 24 * time.Hour

	return ws, nil
}

func NewRepoWorkspaceConfig() (*WorkspaceConfig, error) {

	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	targetDir := currentDir
	rootDir := ""
	if runtime.GOOS == "windows" {
		rootDir = filepath.VolumeName(currentDir) + "\\"
	} else {
		rootDir = "/"
	}

	gitDirFound := false
	for targetDir != rootDir && targetDir != "." && targetDir != "" {
		tryPath := filepath.Join(targetDir, ".git")
		if _, err := os.Stat(tryPath); err == nil {

			gitDirFound = true
			break
		}

		targetDir = filepath.Dir(targetDir)
	}

	if !gitDirFound {
		return nil, fmt.Errorf("unable to find git repository root from current directory '%s'", currentDir)
	}

	ws := NewWorkspaceConfig()
	workspaceDir := filepath.Join(targetDir, ".cast")

	ws.Dir = workspaceDir
	ws.File = filepath.Join(workspaceDir, "workspace.yaml")
	ws.Projects.AutoDiscover.Enabled = true
	ws.Projects.AutoDiscover.Expires = 24 * time.Hour
	ws.Projects.Include = []string{"**"}
	ws.Projects.Cache = []CastfileInfo{}

	return ws, nil
}

func NewGlobalWorkspaceConfig() (*WorkspaceConfig, error) {
	ws := NewWorkspaceConfig()
	userHomeDir, err := paths.UserHomeDir()
	if err != nil {
		return nil, err
	}

	ws.Projects.Cache = []CastfileInfo{}
	homeDir := CastfileInfo{
		Id:      "global",
		Name:    "Global User Tasks",
		Alias:   "global",
		Path:    filepath.Join(userHomeDir, ".castfile"),
		Version: "0.0.0",
		Desc:    "global user tasks",
		Needs:   []Need{},
		Tags:    []string{"global", "user"},
	}

	globalWorkspace := os.Getenv("CAST_GLOBAL_WORKSPACE")
	if globalWorkspace == "" {
		if runtime.GOOS == "windows" {
			globalWorkspace = filepath.Join(userHomeDir, "AppData", "Local", "cast")
		} else {
			globalWorkspace = filepath.Join(userHomeDir, ".local", "share", "cast")
		}
	}

	ws.Projects.Cache = append(ws.Projects.Cache, homeDir)
	ws.Dir = globalWorkspace
	ws.File = filepath.Join(globalWorkspace, "workspace.yaml")

	return ws, nil
}

func (wc *WorkspaceConfig) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})

	if wc.Projects != nil {
		projects, err := wc.Projects.MarshalYAML()
		if err != nil {
			return nil, err
		}
		mapping["projects"] = projects
	}

	if wc.Defaults != nil {
		defaults, err := wc.Defaults.MarshalYAML()
		if err != nil {
			return nil, err
		}
		mapping["defaults"] = defaults
	}

	if wc.Env != nil {
		env, err := wc.Env.MarshalYAML()
		if err != nil {
			return nil, err
		}
		mapping["env"] = env
	}

	if len(wc.Modules) > 0 {
		mapping["modules"] = wc.Modules
	}

	return mapping, nil
}

func (wc *WorkspaceConfig) LoadFile(path string) error {
	bytes, err := os.ReadFile(path)
	wc.File = path
	wc.Dir = filepath.Dir(path)

	if err != nil {
		return err
	}

	return yaml.Unmarshal(bytes, wc)
}

func (wc *WorkspaceConfig) SaveFile(path string) error {
	bytes, err := yaml.Marshal(wc)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, bytes, 0644)
}

func (wc *WorkspaceConfig) DiscoverProjects() {
	if wc.Projects == nil {
		wc.Projects = &ProjectDiscovery{}
	}

	rootDir := os.Getenv("CAST_ROOT_DIR")
	workspaceDir := os.Getenv("CAST_WORKSPACE_DIR")

	if workspaceDir == "" {
		workspaceDir = wc.Dir
	}

	if rootDir == "" {
		rootDir = filepath.Dir(workspaceDir)
	}

	discovery := wc.Projects
	includes := []glob.Glob{}
	excludes := []glob.Glob{}

	for _, pattern := range discovery.Include {
		g := glob.MustCompile(pattern)
		includes = append(includes, g)
	}

	for _, pattern := range discovery.Exclude {
		g := glob.MustCompile(pattern)
		excludes = append(excludes, g)
	}

	wc.Projects.Cache = []CastfileInfo{}

	// walk rootDir and find all projects based on inclusion/exclusion patterns
	filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		for _, ex := range excludes {
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			if ex.Match(relPath) {
				return filepath.SkipDir
			}
		}

		isMatch := false

		if len(includes) == 0 {
			isMatch = true
		} else {
			for _, in := range includes {
				relPath, err := filepath.Rel(rootDir, path)
				if err != nil {
					return err
				}
				if in.Match(relPath) {
					isMatch = true
					break
				}
			}
		}

		if !isMatch {
			return filepath.SkipDir
		}

		tryFiles := []string{
			filepath.Join(rootDir, path, ".castfile"),
			filepath.Join(rootDir, path, "castfile"),
		}

		for _, f := range tryFiles {
			if _, err := os.Stat(f); err == nil {
				info := CastfileInfo{}
				if err := info.LoadFromFile(f); err != nil {
					break
				}

				wc.Projects.Cache = append(wc.Projects.Cache, info)
				break
			}
		}

		return nil
	})

	wc.SaveFile(wc.File)
}

type ProjectMini struct {
	Id    string
	Name  string
	Alias string
}

func (pm *ProjectMini) LoadFromFile(path string) error {
	if pm == nil {
		pm = &ProjectMini{}
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(bytes, &pm)
}

func (pm *ProjectMini) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "id":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			pm.Id = valueNode.Value
		case "name":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			pm.Name = valueNode.Value
		case "alias":
			if valueNode.Kind != yaml.ScalarNode {
				return nil
			}
			pm.Alias = valueNode.Value
		}
	}

	return nil
}

func (wc *WorkspaceConfig) UnmarshalYAML(node *yaml.Node) error {
	if wc == nil {
		wc = &WorkspaceConfig{}
	}

	if node.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "projects":
			var pd ProjectDiscovery
			if err := valueNode.Decode(&pd); err != nil {
				return err
			}
			wc.Projects = &pd
		case "defaults":
			var wd WorkspaceDefaults
			if err := valueNode.Decode(&wd); err != nil {
				return err
			}
			wc.Defaults = &wd
		case "env":
			var env Env
			if err := valueNode.Decode(&env); err != nil {
				return err
			}
			wc.Env = &env
		case "modules":
			if valueNode.Kind != yaml.SequenceNode {
				return nil
			}
			modules := []string{}
			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return nil
				}
				modules = append(modules, item.Value)
			}
			wc.Modules = modules
		}
	}

	return nil
}
