package models

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
)

type ProjectContext struct {
	Project         *Project
	ContextName     string
	Context         context.Context
	Args            []string
	Env             Env
	Outputs         Outputs
	OS              map[string]interface{}
	Git             map[string]string
	Cast            map[string]interface{}
	DisposableFiles map[string]*DisposableFile
	Paths           ProjectPaths
}

type ProjectPaths struct {
	WorkspacePaths
	ProjectFile string
	ProjectDir  string
}

type DisposableFile struct {
	// name of the evironment variable to set
	Name       string
	Path       string
	MustDelete bool
	Deleted    bool
}

func (pc *ProjectContext) Init(rootCtx *WorkspaceContext) error {
	if pc == nil {
		pc = &ProjectContext{}
	}
	pc.Context = rootCtx.Context
	pc.ContextName = rootCtx.ContextName
	pc.Paths.WorkspacePaths = rootCtx.Paths

	pc.Env = Env{}
	pc.Env.Merge(&rootCtx.Env)

	pc.OS = make(map[string]interface{})
	for k, v := range rootCtx.OS {
		pc.OS[k] = v
	}

	pc.Git = make(map[string]string)
	for k, v := range rootCtx.Git {
		if strVal, ok := v.(string); ok {
			pc.Git[k] = strVal
		}
	}

	pc.Outputs = Outputs{}
	pc.Outputs.Merge(rootCtx.Outputs)

	cacheDir := pc.Paths.Cache

	cleanupDir := filepath.Join(cacheDir, "cleanup")

	err := os.MkdirAll(cleanupDir, os.ModePerm)
	if err != nil {
		return err
	}

	files := map[string]string{
		"CAST_ENV":     pc.Env.Get("CAST_ENV"),
		"CAST_PATH":    pc.Env.Get("CAST_PATH"),
		"CAST_OUTPUTS": pc.Env.Get("CAST_OUTPUTS"),
		"CAST_SECRETS": pc.Env.Get("CAST_SECRETS"),
	}

	for name, path := range files {
		dispose := len(path) == 0
		fullPath := path
		pattern := strings.ReplaceAll(strings.ToLower(name), "-", "_")
		if dispose {
			envFile2, err := os.CreateTemp(cleanupDir, pattern)

			if err != nil {
				return errors.Newf("failed to create temporary env file: %w", err)
			}
			err = envFile2.Close()
			if err != nil {
				return errors.Newf("failed to close temporary env file: %w", err)
			}

			fullPath = filepath.Join(cleanupDir, envFile2.Name())
			pc.Env.Set(name, fullPath)
		} else {
			if !filepath.IsAbs(fullPath) {
				fullPath, err = filepath.Abs(fullPath)
				if err != nil {
					return errors.Newf("failed to get absolute path for CAST_ENV: %w", err)
				}
			}

			dir := filepath.Dir(fullPath)
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return errors.Newf("failed to create directories for specified env file: %w", err)
			}

			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				file, err := os.Create(fullPath)
				if err != nil {
					return errors.Newf("failed to create specified env file: %w", err)
				}
				err = file.Close()
				if err != nil {
					return errors.Newf("failed to close specified env file: %w", err)
				}
			}
		}

		pc.DisposableFiles = make(map[string]*DisposableFile)
		pc.DisposableFiles[name] = &DisposableFile{
			Name:       name,
			Path:       fullPath,
			MustDelete: dispose,
			Deleted:    false,
		}
	}

	pc.Paths.WorkspacePaths = rootCtx.Paths
	pc.Paths.ProjectDir = pc.Project.Dir
	pc.Paths.ProjectFile = pc.Project.File

	pc.Cast = make(map[string]interface{})
	for k, v := range rootCtx.Cast {
		pc.Cast[k] = v
	}

	proj := make(map[string]interface{})
	proj["file"] = pc.Paths.ProjectFile
	proj["dir"] = pc.Paths.ProjectDir
	proj["name"] = pc.Project.Name
	proj["version"] = pc.Project.Version
	proj["id"] = pc.Project.Id
	proj["meta"] = pc.Project.Meta
	pc.Cast["project"] = proj

	pc.Env.Set("CAST_PROJECT_DIR", pc.Paths.ProjectDir)
	pc.Env.Set("CAST_PROJECT_FILE", pc.Paths.ProjectFile)
	pc.Env.Set("CAST_PROJECT_NAME", pc.Project.Name)
	pc.Env.Set("CAST_PROJECT_VERSION", pc.Project.Version)
	pc.Env.Set("CAST_PROJECT_ID", pc.Project.Id)

	jsonData, err := json.Marshal(pc.Project.Meta)
	if err != nil {
		return err
	}
	pc.Env.Set("CAST_PROJECT_META", string(jsonData))

	return nil
}

func normalizeEnv(e *Env) error {

	configHome := os.Getenv("XDG_CONFIG_HOME")
	dataHome := os.Getenv("XDG_DATA_HOME")
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	stateHome := os.Getenv("XDG_STATE_HOME")
	binHome := os.Getenv("XDG_BIN_HOME")
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	e.Set("OS_PLATFORM", runtime.GOOS)
	e.Set("OS_ARCH", runtime.GOARCH)

	if runtime.GOOS == "windows" {
		e.Set("OSTYPE", "windows")
		user, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		host, err := os.Hostname()
		if err != nil {
			return err
		}
		shell, ok := os.LookupEnv("SHELL")
		if !ok {
			shell = "powershell.exe"
		}
		e.Set("HOME", user)
		e.Set("HOMEPATH", user)
		e.Set("USER", user)
		e.Set("HOSTNAME", host)
		e.Set("SHELL", shell)
		if len(configHome) == 0 {
			configHome = filepath.Join(user, "AppData", "Roaming")
			e.Set("XDG_CONFIG_HOME", configHome)
		}

		if len(dataHome) == 0 {
			dataHome = filepath.Join(user, "AppData", "Local")
			e.Set("XDG_DATA_HOME", dataHome)
		}

		if len(cacheHome) == 0 {
			cacheHome = filepath.Join(user, "AppData", "Local", "Cache")
			e.Set("XDG_CACHE_HOME", cacheHome)
		}

		if len(stateHome) == 0 {
			stateHome = filepath.Join(user, "AppData", "Local", "State")
			e.Set("XDG_STATE_HOME", stateHome)
		}

		if len(binHome) == 0 {
			binHome = filepath.Join(user, "AppData", "Local", "Programs", "bin")
			e.Set("XDG_BIN_HOME", binHome)
		}

		if len(runtimeDir) == 0 {
			runtimeDir = filepath.Join(user, "AppData", "Local", "Temp")
			e.Set("XDG_RUNTIME_DIR", runtimeDir)
		}
	} else {
		osType := os.Getenv("OSTYPE")
		if len(osType) == 0 {
			osType = runtime.GOOS
			e.Set("OSTYPE", osType)
		}

		user, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		if len(configHome) == 0 {
			configHome = filepath.Join(user, ".config")
			e.Set("XDG_CONFIG_HOME", configHome)
		}

		if len(dataHome) == 0 {
			dataHome = filepath.Join(user, ".local", "share")
			e.Set("XDG_DATA_HOME", dataHome)
		}

		if len(cacheHome) == 0 {
			cacheHome = filepath.Join(user, ".cache")
			e.Set("XDG_CACHE_HOME", cacheHome)
		}

		if len(stateHome) == 0 {
			stateHome = filepath.Join(user, ".local", "state")
			e.Set("XDG_STATE_HOME", stateHome)
		}

		if len(binHome) == 0 {
			binHome = filepath.Join(user, ".local", "bin")
			e.Set("XDG_BIN_HOME", binHome)
		}

		if len(runtimeDir) == 0 {
			id := os.Getuid()
			runtimeDir = filepath.Join("user", "run", fmt.Sprintf("%d", id))
			e.Set("XDG_RUNTIME_DIR", runtimeDir)
		}
	}

	return nil
}
