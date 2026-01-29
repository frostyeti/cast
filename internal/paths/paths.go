package paths

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

var cachedUserHomeDir string

func IsDir(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return fi.IsDir()
	}
	return false
}

func IsFile(path string) bool {
	if fi, err := os.Stat(path); err == nil {
		return !fi.IsDir()
	}
	return false
}

func Resolve(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	if path[0] == '~' {
		homeDir, err := UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[1:]), nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

func ResolvePath(basePath, relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	if relativePath[0] == '~' {
		homeDir, err := UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, relativePath[1:]), nil
	}

	if len(relativePath) == 1 && relativePath == "." {
		return filepath.Abs(basePath)
	}

	if len(relativePath) >= 2 && (relativePath[0:2] == "./" || relativePath[0:2] == ".\\") {
		return filepath.Abs(filepath.Join(basePath, relativePath[2:]))
	}

	return filepath.Abs(filepath.Join(basePath, relativePath))
}

var Common CommonPaths = CommonPaths{}

type CommonPaths struct {
	initialized       bool
	HomeDir           string
	ConfigDir         string
	DataDir           string
	CacheDir          string
	StateDir          string
	BinDirs           []string
	ConfigDirs        []string
	ModulesDirs       []string
	TaskHandlersDirs  []string
	GlobalHandlersDir string
	InventoryDirs     []string
	GlobalHostsDir    string
}

func (cp *CommonPaths) Init(workspacePath string) error {
	if cp.initialized {
		return nil
	}

	cp.initialized = true

	current := workspacePath
	homeDir, err := UserHomeDir()
	if err != nil {
		return err
	}

	cp.HomeDir = homeDir
	cp.ConfigDir, _ = UserConfigDir()
	cp.DataDir, _ = UserDataDir()
	cp.CacheDir, _ = UserCacheDir()
	cp.StateDir, _ = UserStateDir()
	cp.BinDirs = BinDirs(current)
	cp.ConfigDirs = ConfigDirs(current)
	cp.ModulesDirs = ImportDirs(current)
	cp.TaskHandlersDirs = TaskHandlersDirs(current)
	cp.InventoryDirs = IventoryDirs(current)

	dataDir, err := UserDataDir()

	globalHandlersDir := os.Getenv("CAST_GLOBAL_HANDLERS_DIR")
	if globalHandlersDir != "" {
		cp.GlobalHandlersDir = globalHandlersDir
	}

	globalHostsDir := os.Getenv("CAST_GLOBAL_HOSTS_DIR")
	if globalHostsDir != "" {
		cp.GlobalHostsDir = globalHostsDir
	}

	if err == nil {
		if cp.GlobalHandlersDir == "" {
			cp.GlobalHandlersDir = filepath.Join(dataDir, "handlers")
		}
		if cp.GlobalHostsDir == "" {
			cp.GlobalHostsDir = filepath.Join(dataDir, "hosts")
		}
	} else {
		return err
	}

	return nil
}

func UserHomeDir() (string, error) {
	if cachedUserHomeDir != "" {
		return cachedUserHomeDir, nil
	}

	homeDir := os.Getenv("HOME")
	var sudoUser = os.Getenv("SUDO_USER")
	if sudoUser != "" && runtime.GOOS != "windows" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			homeDir = u.HomeDir
		}
	}

	if homeDir == "" {
		usr, err := user.Current()
		if err != nil {
			return "", errors.New("could not determine user home directory: " + err.Error())
		}
		homeDir = usr.HomeDir
	}

	cachedUserHomeDir = homeDir
	return homeDir, nil
}

func ConfigDirs(cwd string) []string {
	localPath := filepath.Join(cwd, ".config", "cast")

	dirs := []string{
		localPath,
		filepath.Join(cwd, ".cast", "etc"),
	}

	configDir, err := UserConfigDir()
	if err == nil {
		if configDir != localPath {
			dirs = append(dirs, configDir)
		}
	}

	return dirs
}

func ImportDirs(cwd string) []string {
	localPath := filepath.Join(cwd, ".local", "share", "cast", "modules")
	dirs := []string{
		localPath,
		filepath.Join(cwd, ".cast", "modules"),
	}

	dataDir, err := UserDataDir()
	if err == nil {
		modulesDir := filepath.Join(dataDir, "modules")
		if modulesDir != localPath {
			dirs = append(dirs, modulesDir)
		}
	}

	return dirs
}

func (c *CommonPaths) ResolveHostsfile(hostsfilePath string) (string, error) {
	if filepath.IsAbs(hostsfilePath) {
		stat, err := os.Stat(hostsfilePath)
		if err != nil {
			return "", err
		}

		if stat.IsDir() {
			return "", errors.New("hostsfile path " + hostsfilePath + " is a directory")
		}

		return hostsfilePath, nil
	}

	if hostsfilePath[0] == '.' {
		resolvedPath, err := filepath.Abs(hostsfilePath)
		if err != nil {
			return "", err
		}

		stat, err := os.Stat(resolvedPath)
		if err != nil {
			return "", err
		}

		if stat.IsDir() {
			return "", errors.New("hostsfile path " + hostsfilePath + " is a directory")
		}

		return resolvedPath, nil
	}

	// trying local file, possibly just the file path as  "hostfile.yaml"
	stat, err := os.Stat(hostsfilePath)
	if err == nil && !stat.IsDir() {
		abs, err := filepath.Abs(hostsfilePath)
		if err != nil {
			return "", err
		}

		return abs, nil
	}

	for _, dir := range c.InventoryDirs {
		fullPath := filepath.Join(dir, hostsfilePath)
		stat, err := os.Stat(fullPath)
		if err == nil && !stat.IsDir() {
			return fullPath, nil
		}
	}

	return "", errors.New("could not resolve hostsfile path: " + hostsfilePath)
}

func (c *CommonPaths) ResolveImport(importPath string) (string, error) {
	if filepath.IsAbs(importPath) {
		stat, err := os.Stat(importPath)
		if err != nil {
			return "", err
		}

		if stat.IsDir() {
			return "", errors.New("import path " + importPath + " is a directory")
		}

		return importPath, nil
	}

	// could be ./ or ../
	if importPath[0] == '.' {
		resolvedPath, err := filepath.Abs(importPath)
		if err != nil {
			return "", err
		}

		stat, err := os.Stat(resolvedPath)
		if err != nil {
			return "", err
		}

		if stat.IsDir() {
			return "", errors.New("import path " + importPath + " is a directory")
		}

		return resolvedPath, nil
	}

	// trying local file, possibly just the file path as  "hostfile.yaml"
	stat, err := os.Stat(importPath)
	if err == nil && !stat.IsDir() {
		abs, err := filepath.Abs(importPath)
		if err != nil {
			return "", err
		}

		return abs, nil
	}

	for _, dir := range c.ModulesDirs {
		fullPath := filepath.Join(dir, importPath)
		stat, err := os.Stat(fullPath)
		if err == nil && !stat.IsDir() {
			return fullPath, nil
		}
	}

	return "", errors.New("could not resolve import path: " + importPath)
}

func IventoryDirs(cwd string) []string {
	dirs := []string{
		filepath.Join(cwd, ".cast", "hosts"),
		filepath.Join(cwd, ".local", "share", "cast", "hosts"),
	}

	globalHostsDir := os.Getenv("CAST_GLOBAL_HOSTS_DIR")
	if globalHostsDir != "" {
		dirs = append(dirs, globalHostsDir)
		return dirs
	}

	dataDir, err := UserDataDir()
	if err == nil {
		dirs = append(dirs, filepath.Join(dataDir, "hosts"))
	}

	return dirs
}

func TaskHandlersDirs(cwd string) []string {
	localPath := filepath.Join(cwd, ".local", "share", "cast", "handlers")
	dirs := []string{
		filepath.Join(cwd, ".cast", "handlers"),
		localPath,
	}

	globalTasksDir := os.Getenv("CAST_GLOBAL_HANDLERS_DIR")
	if globalTasksDir != "" {
		dirs = append(dirs, globalTasksDir)
		return dirs
	}

	dataDir, err := UserDataDir()
	if err == nil {
		handlersDir := filepath.Join(dataDir, "handlers")
		if handlersDir != localPath {
			dirs = append(dirs, handlersDir)
		}
	}

	return dirs
}

func BinDirs(cwd string) []string {
	dirs := []string{}

	miseDataDir := os.Getenv("MISE_DATA_HOME")
	if miseDataDir != "" {
		dirs = append(dirs, filepath.Join(miseDataDir, "shims"))
	}
	denoInstallDir := os.Getenv("DENO_INSTALL")
	bunInstallDir := os.Getenv("BUN_INSTALL")
	cargoHome := os.Getenv("CARGO_HOME")
	home, _ := os.UserHomeDir()

	if runtime.GOOS != "windows" {
		sudoUser := os.Getenv("SUDO_USER")
		home := os.Getenv("HOME")
		if sudoUser != "" {
			u, err := user.Lookup(sudoUser)
			if err == nil {
				homeDir := u.HomeDir
				home = homeDir

			}
		} else if home == "" {
			if usr, err := user.Current(); err == nil {
				homeDir := usr.HomeDir
				home = homeDir
			}
		}

		dirs = append(dirs, filepath.Join(home, ".local", "bin"))
		dirs = append(dirs, filepath.Join(home, ".dotnet", "tools"))
		dirs = append(dirs, filepath.Join(home, "go", "bin"))

		if denoInstallDir != "" {
			dirs = append(dirs, filepath.Join(denoInstallDir, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".deno", "bin"))
		}

		if bunInstallDir != "" {
			dirs = append(dirs, filepath.Join(bunInstallDir, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".bun", "bin"))
		}

		if cargoHome != "" {
			dirs = append(dirs, filepath.Join(cargoHome, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".cargo", "bin"))
		}

		if miseDataDir == "" {
			dataDir := os.Getenv("XDG_DATA_HOME")
			if dataDir == "" {
				dataDir = filepath.Join(home, ".local", "share")
			}
			miseDataDir = filepath.Join(dataDir, "mise")
		}

		dirs = append(dirs, filepath.Join(miseDataDir, "shims"))

		dirs = append(dirs, filepath.Join(cwd, ".local", "bin"))
		dirs = append(dirs, filepath.Join(cwd, ".cast", "bin"))
		dirs = append(dirs, filepath.Join(cwd, "node_modules", ".bin"))
		dirs = append(dirs, filepath.Join(cwd, "bin"))
		dirs = append(dirs, filepath.Join(cwd, ".bin"))
		return dirs
	} else {

		if home == "" {
			if usr, err := user.Current(); err == nil {
				homeDir := usr.HomeDir
				home = homeDir
			}
		}

		dirs = append(dirs, filepath.Join(home, ".local", "bin"))
		chocolateyHome := os.Getenv("ChocolateyInstall")
		programData := os.Getenv("ALLUSERSPROFILE")
		if chocolateyHome != "" {
			dirs = append(dirs, filepath.Join(chocolateyHome, "bin"))
		} else if programData != "" {
			dirs = append(dirs, filepath.Join(programData, "chocolatey", "bin"))
		} else {
			dirs = append(dirs, filepath.Join("C:\\", "ProgramData", "chocolatey", "bin"))
		}

		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		dirs = append(dirs, filepath.Join(localAppData, "Microsoft", "WindowsApps"))
		dirs = append(dirs, filepath.Join(localAppData, "Microsoft", "WinGet", "Links"))
		dirs = append(dirs, filepath.Join(home, "AppData", "Local", "Programs", "bin"))
		dirs = append(dirs, filepath.Join(home, "scoop", "shims"))
		dirs = append(dirs, filepath.Join(home, ".dotnet", "tools"))
		dirs = append(dirs, filepath.Join(home, "go", "bin"))

		if denoInstallDir != "" {
			dirs = append(dirs, filepath.Join(denoInstallDir, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".deno", "bin"))
		}

		if bunInstallDir != "" {
			dirs = append(dirs, filepath.Join(bunInstallDir, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".bun", "bin"))
		}

		if cargoHome != "" {
			dirs = append(dirs, filepath.Join(cargoHome, "bin"))
		} else {
			dirs = append(dirs, filepath.Join(home, ".cargo", "bin"))
		}

		if miseDataDir == "" {
			dataDir := os.Getenv("XDG_DATA_HOME")
			if dataDir == "" {
				dataDir = filepath.Join(home, ".local", "share")
			}
			miseDataDir = filepath.Join(dataDir, "mise")
		}

		dirs = append(dirs, filepath.Join(miseDataDir, "shims"))

		dirs = append(dirs, filepath.Join(cwd, ".local", "bin"))
		dirs = append(dirs, filepath.Join(cwd, ".cast", "bin"))
		dirs = append(dirs, filepath.Join(cwd, "node_modules", ".bin"))
		dirs = append(dirs, filepath.Join(cwd, "bin"))
		dirs = append(dirs, filepath.Join(cwd, ".bin"))

		return dirs
	}
}

func UserConfigDir() (string, error) {
	configDir := os.Getenv("CAST_CONFIG_HOME")
	if configDir != "" {
		return configDir, nil
	}

	configDir = os.Getenv("CAST_CONFIG_HOME")
	if configDir != "" {
		return filepath.Join(configDir, "cast"), nil
	}

	userDir, err := UserHomeDir()
	if err == nil {
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData != "" {
				return filepath.Join(appData, "cast"), nil
			}

			return filepath.Join(userDir, "AppData", "Roaming", "cast"), nil
		} else {
			return filepath.Join(userDir, ".config", "cast"), nil
		}
	}

	return "", errors.New("Could not determine user config directory: " + err.Error())
}

func UserDataDir() (string, error) {
	dataDir := os.Getenv("CAST_DATA_HOME")
	if dataDir != "" {
		return dataDir, nil
	}

	dataDir = os.Getenv("XDG_DATA_HOME")
	if dataDir != "" {
		return filepath.Join(dataDir, "cast"), nil
	}

	homeDir, err := UserHomeDir()

	if runtime.GOOS == "windows" {
		dataDir = os.Getenv("LOCALAPPDATA")
		if dataDir != "" {
			return filepath.Join(dataDir, "cast", "data"), nil
		}

		return filepath.Join(homeDir, "AppData", "Local", "cast", "data"), nil
	}

	if err == nil {
		return filepath.Join(homeDir, ".local", "share", "cast"), nil
	}

	return "", errors.New("could not determine user data directory")
}

func UserCacheDir() (string, error) {
	cacheDir := os.Getenv("CAST_CACHE_HOME")
	if cacheDir != "" {
		return cacheDir, nil
	}

	cacheDir = os.Getenv("XDG_CACHE_HOME")
	if cacheDir != "" {
		return filepath.Join(cacheDir, "cast"), nil
	}

	homeDir, err := UserHomeDir()

	if runtime.GOOS == "windows" {
		cacheDir = os.Getenv("LOCALAPPDATA")
		if cacheDir != "" {
			return filepath.Join(cacheDir, "cast", "cache"), nil
		}

		if err != nil {
			return "", errors.New("Could not determine user cache directory: " + err.Error())
		}

		return filepath.Join(homeDir, "AppData", "Local", "cast", "cache"), nil
	}

	if err == nil {
		return filepath.Join(homeDir, ".cache", "cast"), nil
	}

	return "", errors.New("Could not determine user cache directory: " + err.Error())
}

func UserStateDir() (string, error) {
	stateDir := os.Getenv("CAST_STATE_HOME")
	if stateDir != "" {
		return stateDir, nil
	}

	stateDir = os.Getenv("XDG_STATE_HOME")
	if stateDir != "" {
		return filepath.Join(stateDir, "cast"), nil
	}

	if runtime.GOOS == "windows" {
		stateDir = os.Getenv("LOCALAPPDATA")
		if stateDir != "" {
			return filepath.Join(stateDir, "State", "cast"), nil
		}
	} else {
		stateDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(stateDir, ".local", "state", "cast"), nil
		}
	}
	return "", errors.New("could not determine user state directory")
}
