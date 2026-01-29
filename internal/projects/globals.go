package projects

import (
	"iter"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/types"
)

var globalScope = &Scope{
	values: make(map[string]any),
	keys:   []string{},
}

var globalEnv = types.NewEnv()

var projectCache = make(map[string]*Project)

type Scope struct {
	values map[string]any
	keys   []string
}

func NewScope() *Scope {
	return &Scope{
		values: make(map[string]any),
		keys:   []string{},
	}
}

func (s *Scope) Get(key string) (any, bool) {
	val, ok := s.values[key]
	return val, ok
}

func (s *Scope) Set(key string, value any) {
	if _, exists := s.values[key]; !exists {
		s.keys = append(s.keys, key)
	}
	s.values[key] = value
}

func (s *Scope) Has(key string) bool {
	_, ok := s.values[key]
	return ok
}

func (s *Scope) Keys() []string {
	return s.keys
}

func (s *Scope) Values() []any {
	values := make([]any, 0, len(s.keys))
	for _, key := range s.keys {
		values = append(values, s.values[key])
	}
	return values
}

func (s *Scope) ToMap() map[string]any {
	return s.values
}

func (s *Scope) Clone() *Scope {
	clone := &Scope{
		values: make(map[string]any, len(s.values)),
		keys:   make([]string, len(s.keys)),
	}
	copy(clone.keys, s.keys)
	for k, v := range s.values {
		clone.values[k] = v
	}
	return clone
}

func (s *Scope) Iter() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for _, key := range s.keys {
			if !yield(key, s.values[key]) {
				return
			}
		}
	}
}

func userIsAdmin() bool {
	if runtime.GOOS == "windows" {
		// On Windows, check for membership in the Administrators group
		currentUser, err := user.Current()
		if err != nil {
			return false
		}
		groups, err := currentUser.GroupIds()
		if err != nil {
			return false
		}
		for _, gid := range groups {
			group, err := user.LookupGroupId(gid)
			if err == nil && group.Name == "Administrators" {
				return true
			}
		}
		return false
	} else {
		// On Unix-like systems, check if UID is 0 (root)
		return os.Geteuid() == 0
	}
}

func init() {
	isWindows := runtime.GOOS == "windows"
	isLinux := runtime.GOOS == "linux"
	isDarwin := runtime.GOOS == "darwin"
	home, err := os.UserHomeDir()
	if err != nil {
		u, err := user.Current()
		if err == nil {
			home = u.HomeDir
		}
	}
	sudoUser := os.Getenv("SUDO_USER")
	sudoUserHome := ""
	if sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			sudoUserHome = u.HomeDir
		}
	}

	globalScope.Set("home", home)
	globalScope.Set("os", runtime.GOOS)
	globalScope.Set("arch", runtime.GOARCH)
	globalScope.Set("num_cpu", runtime.NumCPU())
	globalScope.Set("windows", runtime.GOOS == "windows")
	globalScope.Set("linux", runtime.GOOS == "linux")
	globalScope.Set("darwin", runtime.GOOS == "darwin")
	globalScope.Set("sudo_user", sudoUser)
	globalScope.Set("sudo_user_home", sudoUserHome)
	globalScope.Set("is_admin", userIsAdmin())

	cwd, err := os.Getwd()
	if err == nil {
		globalScope.Set("cwd", cwd)
	}

	ge := globalEnv
	for _, envVar := range os.Environ() {
		kv := strings.SplitN(envVar, "=", 2)
		key := kv[0]
		value := ""
		if len(kv) > 1 {
			value = kv[1]
		}
		ge.Set(key, value)
	}

	globalEnv.Set("OS", runtime.GOOS)
	globalEnv.Set("ARCH", runtime.GOARCH)
	globalEnv.Set("WINDOWS", strconv.FormatBool(isWindows))
	globalEnv.Set("LINUX", strconv.FormatBool(isLinux))
	globalEnv.Set("DARWIN", strconv.FormatBool(isDarwin))
	globalEnv.Set("NUM_CPU", strconv.Itoa(runtime.NumCPU()))
	globalEnv.Set("HOME", home)
	globalEnv.Set("CWD", cwd)
	globalEnv.Set("SUDO_USER_HOME", sudoUserHome)
	globalEnv.Set("IS_ADMIN", strconv.FormatBool(userIsAdmin()))

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" && home != "" {
		xdgDataHome = filepath.Join(home, ".local", "share")
	}

	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" && home != "" {
		xdgConfigHome = filepath.Join(home, ".config")
	}

	xdgCacheHome := os.Getenv("XDG_CACHE_HOME")
	if xdgCacheHome == "" && home != "" {
		xdgCacheHome = filepath.Join(home, ".cache")
	}

	xdgBinHome := os.Getenv("XDG_BIN_HOME")
	if xdgBinHome == "" {
		if sudoUserHome != "" {
			xdgBinHome = filepath.Join(sudoUserHome, ".local", "bin")
		} else if home != "" {
			xdgBinHome = filepath.Join(home, ".local", "bin")
		}
	}

	globalEnv.Set("CAST_XDG_DATA_HOME", xdgDataHome)
	globalEnv.Set("CAST_XDG_CONFIG_HOME", xdgConfigHome)
	globalEnv.Set("CAST_XDG_CACHE_HOME", xdgCacheHome)
	globalEnv.Set("CAST_XDG_BIN_HOME", xdgBinHome)
}
