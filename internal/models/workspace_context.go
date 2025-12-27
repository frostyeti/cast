package models

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/go/env"
	"github.com/go-git/go-git/v5"
)

type WorkspaceContext struct {
	Env         Env
	OS          map[string]interface{}
	Git         map[string]any
	Outputs     Outputs
	ContextName string
	Context     context.Context
	Cast        map[string]interface{}
	Paths       WorkspacePaths
}

type WorkspacePaths struct {
	Workspace string
	Git       string
	Root      string
	Vendor    string
	Modules   string
	Inventory string
	Cache     string
}

func (rc *WorkspaceContext) Init() error {
	if rc == nil {
		rc = &WorkspaceContext{
			Context:     context.TODO(),
			ContextName: "default",
		}
	}

	rc.Cast = make(map[string]interface{})
	rc.Outputs = NewOutputs()
	rc.OS = make(map[string]interface{})
	rc.Git = make(map[string]any)
	menv := Env{}
	for k, v := range env.All() {
		menv.Set(k, v)
	}

	normalizeEnv(&menv)

	o := rc.OS
	devNull := "/dev/null"
	eol := "\n"
	dirSep := "/"
	pathSep := ":"

	tempDir := os.TempDir()
	if menv.Has("CAST_TEMP_DIR") {
		tempDir = menv.Get("CAST_TEMP_DIR")
	}
	menv.Set("CAST_TEMP_DIR", tempDir)

	if runtime.GOOS == "windows" {
		devNull = "NUL"
		eol = "\r\n"
		dirSep = "\\"
		pathSep = ";"
	}

	o["dev_null"] = devNull
	o["eol"] = eol
	o["dir_sep"] = dirSep
	o["path_sep"] = pathSep
	o["temp_dir"] = tempDir
	o["platform"] = runtime.GOOS
	o["arch"] = runtime.GOARCH
	o["is_windows"] = runtime.GOOS == "windows"
	o["is_linux"] = runtime.GOOS == "linux"
	o["is_darwin"] = runtime.GOOS == "darwin"
	menv.Set("OS_PLATFORM", runtime.GOOS)
	menv.Set("OS_ARCH", runtime.GOARCH)
	menv.Set("OS_TEMP_DIR", tempDir)
	menv.Set("OS_IS_WINDOWS", strconv.FormatBool(runtime.GOOS == "windows"))
	menv.Set("OS_IS_LINUX", strconv.FormatBool(runtime.GOOS == "linux"))
	menv.Set("OS_IS_DARWIN", strconv.FormatBool(runtime.GOOS == "darwin"))

	rc.Env = menv
	rc.OS = o

	cd, err := os.Getwd()
	if err != nil {
		cd = ""
	}

	targetDir := cd
	for targetDir != "" && targetDir != "/" {
		gitDir := filepath.Join(targetDir, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			rc.Paths.Git = gitDir
			break
		}
	}

	home := ""

	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		sudoUserObj, err := user.Lookup(sudoUser)
		if err == nil {
			home = sudoUserObj.HomeDir
		} else {
			return errors.Newf("Unable to lookup home directory for SUDO_USER '%s'", sudoUser)
		}
	}

	home2, err := os.UserHomeDir()
	if err != nil {
		user, err2 := user.Current()
		if err2 != nil {
			return errors.Newf("Unable to get home directory for current user : %v", err2)
		}

		home2 = user.HomeDir
	} else {
		home = home2
	}

	workspaceDir := rc.Env.Get("CAST_WORKSPACE")
	if workspaceDir == "" {
		targetDir = cd
		for targetDir != "" && targetDir != "/" {
			castDir := filepath.Join(targetDir, ".cast")
			info, err := os.Stat(castDir)
			if err == nil && info.IsDir() {
				workspaceDir = targetDir
				break
			}
			targetDir = filepath.Dir(targetDir)
		}

		if workspaceDir == "" {
			globalWorkspace := os.Getenv("CAST_GLOBAL_WORKSPACE")
			if globalWorkspace != "" {
				workspaceDir = globalWorkspace
			}

			if workspaceDir == "" {

				if runtime.GOOS == "windows" {
					workspaceDir = filepath.Join(home, "AppData", "Local", "cast")
				} else {
					workspaceDir = filepath.Join(home, ".local", "share", "cast")
				}
			}
		}

		rc.Env.Set("CAST_WORKSPACE", workspaceDir)
		rc.Cast["workspace"] = workspaceDir
		rc.Paths.Workspace = workspaceDir
	}

	if rc.Paths.Git != "" {
		rc.Paths.Root = filepath.Dir(rc.Paths.Git)
	} else if rc.Paths.Workspace != "" {
		if strings.HasPrefix(rc.Paths.Workspace, home) {
			rc.Paths.Root = cd
		} else if strings.HasSuffix(rc.Paths.Workspace, ".cast") {
			rc.Paths.Root = filepath.Dir(rc.Paths.Workspace)
		} else {
			rc.Paths.Root = rc.Paths.Workspace
		}
	} else {
		rc.Paths.Root = cd
	}
	rc.Cast["root"] = rc.Paths.Root
	rc.Env.Set("CAST_ROOT_DIR", rc.Paths.Root)

	vendorDir := rc.Env.Get("CAST_VENDOR_DIR")
	modulesDir := rc.Env.Get("CAST_MODULES_DIR")
	inventoryDir := rc.Env.Get("CAST_INVENTORY_DIR")
	cacheDir := rc.Env.Get("CAST_CACHE_DIR")
	if cacheDir == "" {
		cacheDir = filepath.Join(rc.Paths.Workspace, "cache")
	}

	if vendorDir == "" {
		vendorDir = filepath.Join(rc.Paths.Workspace, "vendor")
	}

	if modulesDir == "" {
		modulesDir = filepath.Join(rc.Paths.Workspace, "modules")
	}

	if inventoryDir == "" {
		inventoryDir = filepath.Join(rc.Paths.Workspace, "inventory")
	}

	if !filepath.IsAbs(vendorDir) {
		updated, err := filepath.Abs(vendorDir)
		if err != nil {
			return errors.Newf("unable to resolve absolute path for vendor dir '%s': %v", vendorDir, err)
		}
		vendorDir = updated
	}

	if !filepath.IsAbs(modulesDir) {
		updated, err := filepath.Abs(modulesDir)
		if err != nil {
			return errors.Newf("unable to resolve absolute path for modules dir '%s': %v", modulesDir, err)
		}
		modulesDir = updated
	}

	if !filepath.IsAbs(inventoryDir) {
		updated, err := filepath.Abs(inventoryDir)
		if err != nil {
			return errors.Newf("unable to resolve absolute path for inventory dir '%s': %v", inventoryDir, err)
		}
		inventoryDir = updated
	}

	if !filepath.IsAbs(cacheDir) {
		updated, err := filepath.Abs(cacheDir)
		if err != nil {
			return errors.Newf("unable to resolve absolute path for cache dir '%s': %v", cacheDir, err)
		}
		cacheDir = updated
	}

	rc.Env.Set("CAST_VENDOR_DIR", vendorDir)
	rc.Env.Set("CAST_MODULES_DIR", modulesDir)
	rc.Env.Set("CAST_INVENTORY_DIR", inventoryDir)
	rc.Env.Set("CAST_CACHE_DIR", cacheDir)

	rc.Paths.Vendor = vendorDir
	rc.Paths.Modules = modulesDir
	rc.Paths.Inventory = inventoryDir
	rc.Paths.Cache = cacheDir
	rc.Cast["vendor"] = vendorDir
	rc.Cast["modules"] = modulesDir
	rc.Cast["inventory"] = inventoryDir
	rc.Cast["cache"] = cacheDir

	rc.Git = make(map[string]any)

	if rc.Paths.Git != "" {
		gitRepo, err := git.PlainOpen(rc.Paths.Root)
		if err == nil {
			remote, err := gitRepo.Remote("origin")
			if err == nil && len(remote.Config().URLs) > 0 {
				rc.Git["remote"] = remote.Config().URLs[0]
			}

			// get current branch
			// get current ref
			headRef, err := gitRepo.Head()
			if err == nil && headRef != nil {
				rc.Git["commit"] = headRef.Hash().String()
				rc.Git["ref"] = headRef.Name().String()
				rc.Git["short_hash"] = headRef.Hash().String()[:7]
				rc.Git["short_ref"] = headRef.Name().Short()
				rc.Git["is_branch"] = headRef.Name().IsBranch()
				rc.Git["is_tag"] = headRef.Name().IsTag()
				rc.Git["ref"] = headRef.String()

				rc.Env.Set("GIT_COMMIT", headRef.Hash().String())
				rc.Env.Set("GIT_REF", headRef.Name().String())
				rc.Env.Set("GIT_SHORT_HASH", headRef.Hash().String()[:7])
				rc.Env.Set("GIT_SHORT_REF", headRef.Name().Short())
				rc.Env.Set("GIT_IS_BRANCH", strconv.FormatBool(headRef.Name().IsBranch()))
				rc.Env.Set("GIT_IS_TAG", strconv.FormatBool(headRef.Name().IsTag()))
				rc.Env.Set("GIT_REF", headRef.String())

				isMaster := headRef.Name().Short() == "master"
				isMain := headRef.Name().Short() == "main"
				isPrimary := isMaster || isMain
				rc.Git["is_primary"] = isPrimary
				rc.Git["is_main"] = isMain
				rc.Git["is_master"] = isMaster

				rc.Env.Set("GIT_IS_PRIMARY", strconv.FormatBool(isPrimary))
				rc.Env.Set("GIT_IS_MAIN", strconv.FormatBool(isMain))
				rc.Env.Set("GIT_IS_MASTER", strconv.FormatBool(isMaster))
			}

			// get auth info for latest commit
			logIter, err := gitRepo.Log(&git.LogOptions{From: headRef.Hash()})
			if err == nil {
				commit, err := logIter.Next()
				if err == nil && commit != nil {
					rc.Git["author"] = map[string]any{
						"name":  commit.Author.Name,
						"email": commit.Author.Email,
						"when":  commit.Author.When,
					}
					rc.Env.Set("GIT_AUTHOR_NAME", commit.Author.Name)
					rc.Env.Set("GIT_AUTHOR_EMAIL", commit.Author.Email)
					rc.Env.Set("GIT_AUTHOR_DATE", commit.Author.When.Format("2006-01-02T15:04:05Z07:00"))

					rc.Git["committer"] = map[string]any{
						"name":  commit.Committer.Name,
						"email": commit.Committer.Email,
						"when":  commit.Committer.When,
					}

					rc.Env.Set("GIT_COMMITTER_NAME", commit.Committer.Name)
					rc.Env.Set("GIT_COMMITTER_EMAIL", commit.Committer.Email)
					rc.Env.Set("GIT_COMMITTER_DATE", commit.Committer.When.Format("2006-01-02T15:04:05Z07:00"))
				}
			}

		}
	}

	return nil
}
