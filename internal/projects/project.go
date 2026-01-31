package projects

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"github.com/frostyeti/cast/internal/eval"
	"github.com/frostyeti/cast/internal/id"
	"github.com/frostyeti/cast/internal/paths"
	"github.com/frostyeti/cast/internal/types"
	"github.com/frostyeti/go/dotenv"
	"github.com/frostyeti/go/env"
	"github.com/gobwas/glob"
)

type Project struct {
	Id             string
	Env            *types.Env
	Secrets        *types.Env
	Tasks          types.TaskMap
	Hosts          map[string]HostInfo
	Scope          *Scope
	ContextName    string
	Schema         types.Project
	File           string
	Dir            string
	CastDir        string
	imported       map[string]types.Module
	importedOrder  []string
	init           bool
	cleanupEnv     bool
	cleanupPath    bool
	cleanupOutputs bool
	Workspace      map[string]*ProjectInfo
}

type ProjectInfo struct {
	Alias   string
	Path    string
	Rel     string
	Project *Project
}

func (p *Project) InitWorkspace() error {
	if p.Schema.Workspace == nil {
		return nil
	}

	p.Workspace = make(map[string]*ProjectInfo)

	for alias, path := range p.Schema.Workspace.Aliases {
		proj := &ProjectInfo{}
		proj.Alias = alias
		proj.Path = path
		p.Workspace[alias] = proj
	}

	excludes := []glob.Glob{}
	includes := []glob.Glob{}

	if !slices.Contains(p.Schema.Workspace.Exclude, "**/node_modules/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "**/node_modules/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "node_modules/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "node_modules/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "**/bin/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "**/bin/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "bin/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "bin/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "**/obj/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "**/obj/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "obj/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "obj/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, "**/.git/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, "**/.git/**")
	}

	if !slices.Contains(p.Schema.Workspace.Exclude, ".git/**") {
		p.Schema.Workspace.Exclude = append(p.Schema.Workspace.Exclude, ".git/**")
	}

	for _, pattern := range p.Schema.Workspace.Exclude {
		if strings.HasSuffix(pattern, "/**") {
			excludes = append(excludes, glob.MustCompile(pattern[:len(pattern)-3]))
		}

		excludes = append(excludes, glob.MustCompile(pattern))
	}

	for _, pattern := range p.Schema.Workspace.Include {
		if strings.HasSuffix(pattern, "/**") {
			includes = append(includes, glob.MustCompile(pattern[:len(pattern)-3]))
		}

		includes = append(includes, glob.MustCompile(pattern))
	}

	dir := p.Dir

	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {

		if !d.IsDir() {
			name := d.Name()
			if name == "castfile" || name == ".castfile" || name == "cast.yaml" || name == "cast.yml" {
				proj := &ProjectInfo{}
				proj.Path = path
				dir := filepath.Dir(path)
				basename := filepath.Base(dir)
				proj.Alias = basename
				relPath, _ := filepath.Rel(dir, path)
				proj.Rel = relPath
				_, ok := p.Workspace[proj.Alias]
				if ok {
					// write warning to stdout, use yelling style
					os.Stdout.WriteString("\x1b[31mWARNING: workspace alias " + proj.Alias + " already exists, skipping " + path + "\n\x1b[0m")
				}

				p.Workspace[proj.Alias] = proj
			}

			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		skip := false
		for _, ex := range excludes {
			if ex.Match(relPath) {
				skip = true
				break
			}
		}

		if skip {
			return filepath.SkipDir
		}

		if len(includes) > 0 {
			match := false
			for _, inc := range includes {
				if inc.Match(relPath) {
					match = true
					break
				}
			}

			if !match {
				return filepath.SkipDir
			}
		}

		return nil
	})

	return nil
}

func (p *Project) LoadFromYaml(file string) error {
	err := p.Schema.ReadFromYaml(file)
	if err != nil {
		return err
	}
	p.File = file
	p.Dir = filepath.Dir(file)
	p.Hosts = make(map[string]HostInfo)
	p.Env = types.NewEnv()
	return nil
}

func (p *Project) Init() error {
	if p.init {
		return nil
	}
	p.init = true

	if p.ContextName == "" {
		p.ContextName = env.Get("CAST_CONTEXT")
		if p.ContextName == "" {
			p.ContextName = "default"
		}
	}

	p.imported = make(map[string]types.Module)

	targetDir := p.Dir
	if p.Schema.Workspace == nil {
		for targetDir != "/" && targetDir != "" {
			tryDirs := []string{
				filepath.Join(targetDir, ".cast"),
				filepath.Join(targetDir, ".config", "cast"),
			}

			found := false
			for _, dir := range tryDirs {
				info, err := os.Stat(dir)
				if err == nil && info.IsDir() {
					p.CastDir = dir
					found = true
					break
				}
			}

			if !found {
				if _, err := os.Stat(filepath.Join(targetDir, ".git")); err == nil {
					p.CastDir = filepath.Join(targetDir, ".cast")
					found = true
				}
			}

			if found {
				break
			}

			targetDir = filepath.Dir(targetDir)
		}

		if p.CastDir == "" {
			data, err := paths.UserDataDir()
			if err == nil {
				p.CastDir = filepath.Join(data, "cast")
			} else {
				return errors.Newf("unable to get the cast directory")
			}
		}
	} else {
		p.CastDir = filepath.Join(p.Dir, ".config", "cast")
		if _, err := os.Stat(p.CastDir); os.IsNotExist(err) {
			p.CastDir = filepath.Join(p.Dir, ".cast")
		}
	}

	err := resolveModules(p, p.Schema.Imports)
	if err != nil {
		return err
	}

	err = setupEnv(p)
	if err != nil {
		return err
	}

	p.Scope = globalScope.Clone()
	p.Scope.Set("env", env.All())
	p.Scope.Set("cast", map[string]any{
		"meta": p.Schema.Meta.ToMap(),
	})

	err = loadModules(p)
	if err != nil {
		return err
	}

	scope := p.Scope.ToMap()

	substitution := true
	if p.Schema.Config != nil && p.Schema.Config.Substitution != nil {
		substitution = *p.Schema.Config.Substitution
	}

	if p.Schema.Inventory != nil && len(p.Schema.Inventory.Hosts) > 0 {
		defaultsMap := p.Schema.Inventory.Defaults

		for k, h := range p.Schema.Inventory.Hosts {
			defaultsName := h.Defaults
			if defaultsName == "" {
				defaultsName = "default"
			}

			d, ok := defaultsMap[defaultsName]
			if ok {
				if (h.User == nil || *h.User == "") && d.User != nil {
					h.User = d.User
				}

				if (h.Port == nil || *h.Port == 0) && d.Port != nil {
					h.Port = d.Port
				}

				if (h.IdentityFile == nil || *h.IdentityFile == "") && d.IdentityFile != nil {
					h.IdentityFile = d.IdentityFile
				}

				if (h.Password == nil || *h.Password == "") && d.Password != nil {
					h.Password = d.Password
				}

				if h.OS == nil && d.OS != nil {
					h.OS = d.OS
				} else if h.OS != nil && d.OS != nil {
					if h.OS.Version == "" && d.OS.Version != "" {
						h.OS.Version = d.OS.Version
					}

					if h.OS.Arch == "" && d.OS.Arch != "" {
						h.OS.Arch = d.OS.Arch
					}

					if h.OS.Family == "" && d.OS.Family != "" {
						h.OS.Family = d.OS.Family
					}

					if h.OS.Variant == "" && d.OS.Variant != "" {
						h.OS.Variant = d.OS.Variant
					}
				}

				if h.Meta == nil && d.Meta != nil {
					h.Meta = d.Meta
				} else if h.Meta != nil && d.Meta != nil {
					for k, v := range d.Meta.ToMap() {
						if _, exists := h.Meta.ToMap()[k]; !exists {
							h.Meta.Set(k, v)
						}
					}
				}

				for _, g := range d.Tags {
					if !slices.Contains(h.Tags, g) {
						h.Tags = append(h.Tags, g)
					}
				}
			} else {
				if defaultsName != "default" {
					return errors.Newf("host %s references undefined defaults %s", h.Host, h.Defaults)
				}
			}
			if strings.ContainsRune(h.Host, '{') {
				v, err := eval.EvalAsString(h.Host, scope)
				if err != nil {
					return err
				}
				h.Host = v
			}

			if h.Password != nil && strings.ContainsRune(*h.Password, '{') {
				v, err := eval.EvalAsString(*h.Password, scope)
				if err != nil {
					return err
				}
				*h.Password = v
			}

			if h.Password != nil && strings.ContainsRune(*h.Password, '$') {
				v, err := env.Expand(*h.Password, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
				if err != nil {
					return err
				}
				*h.Password = v
			}

			if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '{') {
				v, err := eval.EvalAsString(*h.IdentityFile, scope)
				if err != nil {
					return err
				}
				*h.IdentityFile = v
			}

			if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '$') {
				v, err := env.Expand(*h.IdentityFile, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
				if err != nil {
					return err
				}
				*h.IdentityFile = v
			}

			for _, h2 := range p.Hosts {
				if h2.Host == h.Host {
					return errors.Newf("duplicate host entry for host %s", h.Host)
				}
			}

			port := uint(22)
			if h.Port != nil && *h.Port > 0 {
				port = *h.Port
			}

			user := ""
			if h.User != nil {
				user = *h.User
			}

			password := ""
			if h.Password != nil {
				password = *h.Password
			}

			identityFile := ""
			if h.IdentityFile != nil {
				identityFile = *h.IdentityFile
			}

			osInfo := types.OsInfo{}
			if h.OS != nil {
				osInfo = *h.OS
			}

			meta := types.NewMeta()
			if h.Meta != nil {
				meta = h.Meta
			}

			p.Hosts[k] = HostInfo{
				Host:         h.Host,
				Port:         port,
				User:         user,
				Password:     password,
				IdentityFile: identityFile,
				OS:           osInfo,
				Meta:         *meta,
				Tags:         h.Tags,
			}

			if _, ok := p.Hosts[h.Host]; !ok {
				p.Hosts[h.Host] = p.Hosts[k]
			}
		}
	}

	for _, task := range p.Schema.Tasks.Values() {
		if task.Id == "" {
			task.Id = id.Convert(task.Name)
		}

		p.Tasks.Set(&task)
	}

	for _, task := range p.Tasks.Values() {
		extends := task.Extends
		if extends != nil {
			baseTask, ok := p.Tasks.Get(*extends)
			if !ok {
				return errors.Newf("task %s extends undefined task %s", task.Name, *extends)
			}

			if task.Desc == nil && baseTask.Desc != nil {
				task.Desc = baseTask.Desc
			}

			if task.Uses == nil && baseTask.Uses != nil {
				task.Uses = baseTask.Uses
			}

			if task.Cwd == nil && baseTask.Cwd != nil {
				task.Cwd = baseTask.Cwd
			}

			if task.Run == nil && baseTask.Run != nil {
				task.Run = baseTask.Run
			}

			if task.Help == nil && baseTask.Help != nil {
				task.Help = baseTask.Help
			}

			if task.Timeout == nil && baseTask.Timeout != nil {
				task.Timeout = baseTask.Timeout
			}

			if len(task.Needs) == 0 && len(baseTask.Needs) > 0 {
				task.Needs = baseTask.Needs
			}

			if task.Hooks == nil && baseTask.Hooks != nil {
				task.Hooks = baseTask.Hooks
			}

			if len(task.Hosts) == 0 && len(baseTask.Hosts) > 0 {
				task.Hosts = baseTask.Hosts
			} else {
				hosts := make([]string, 0)
				for _, h := range baseTask.Hosts {
					hosts = append(hosts, h)
				}
				for _, h := range task.Hosts {
					if !slices.Contains(hosts, h) {
						hosts = append(hosts, h)
					}
				}
				task.Hosts = hosts
			}

			if task.Force == nil && baseTask.Force != nil {
				task.Force = baseTask.Force
			}

			if len(task.DotEnv) == 0 && len(baseTask.DotEnv) > 0 {
				task.DotEnv = baseTask.DotEnv
			} else if len(task.DotEnv) > 0 && len(baseTask.DotEnv) > 0 {
				dotenv := []string{}

				for _, de := range baseTask.DotEnv {
					dotenv = append(dotenv, de)
				}

				for _, de := range task.DotEnv {
					replaceIndex := -1
					for i, existingDe := range dotenv {
						if de == existingDe {
							replaceIndex = i
						}
					}

					if replaceIndex > -1 {
						dotenv[replaceIndex] = de
					} else {
						dotenv = append(dotenv, de)
					}
				}

				task.DotEnv = dotenv
			}

			e := baseTask.Env.Clone()
			for k, v := range task.Env.Iter() {
				e.Set(k, v)
			}
			task.Env = e

			meta := baseTask.With.Clone()
			for _, k := range task.With.Keys() {
				v, _ := task.With.Get(k)
				meta.Set(k, v)
			}
			task.With = meta
		}
	}

	return nil
}

func loadModules(p *Project) error {

	if len(p.importedOrder) == 0 {
		return nil
	}

	scope := p.Scope.ToMap()

	substitution := true
	if p.Schema.Config != nil && p.Schema.Config.Substitution != nil {
		substitution = *p.Schema.Config.Substitution
	}

	if len(p.imported) > 0 {
		for _, path := range p.importedOrder {
			mod := p.imported[path]

			if mod.Inventory != nil && len(mod.Inventory.Hosts) > 0 {
				defaultsMap := mod.Inventory.Defaults

				for _, hostName := range mod.Inventory.HostOrder {
					h := mod.Inventory.Hosts[hostName]
					defaultsName := h.Defaults
					if defaultsName == "" {
						defaultsName = "default"
					}

					d, ok := defaultsMap[defaultsName]
					if ok {
						if (h.User == nil || *h.User == "") && d.User != nil {
							h.User = d.User
						}

						if (h.Port == nil || *h.Port == 0) && d.Port != nil {
							h.Port = d.Port
						}

						if (h.IdentityFile == nil || *h.IdentityFile == "") && d.IdentityFile != nil {
							h.IdentityFile = d.IdentityFile
						}

						if (h.Password == nil || *h.Password == "") && d.Password != nil {
							h.Password = d.Password
						}

						if h.OS == nil && d.OS != nil {
							h.OS = d.OS
						} else if h.OS != nil && d.OS != nil {
							if h.OS.Version == "" && d.OS.Version != "" {
								h.OS.Version = d.OS.Version
							}

							if h.OS.Arch == "" && d.OS.Arch != "" {
								h.OS.Arch = d.OS.Arch
							}

							if h.OS.Family == "" && d.OS.Family != "" {
								h.OS.Family = d.OS.Family
							}

							if h.OS.Variant == "" && d.OS.Variant != "" {
								h.OS.Variant = d.OS.Variant
							}
						}

						if h.Meta == nil && d.Meta != nil {
							h.Meta = d.Meta
						} else if h.Meta != nil && d.Meta != nil {
							for k, v := range d.Meta.ToMap() {
								if _, exists := h.Meta.ToMap()[k]; !exists {
									h.Meta.Set(k, v)
								}
							}
						}

						for _, g := range d.Tags {
							if !slices.Contains(h.Tags, g) {
								h.Tags = append(h.Tags, g)
							}
						}
					} else {
						if defaultsName != "default" {
							return errors.Newf("host %s references undefined defaults %s", h.Host, h.Defaults)
						}
					}

					if strings.ContainsRune(h.Host, '{') {
						v, err := eval.EvalAsString(h.Host, scope)
						if err != nil {
							return err
						}
						h.Host = v
					}

					if h.Password != nil && strings.ContainsRune(*h.Password, '{') {
						v, err := eval.EvalAsString(*h.Password, scope)
						if err != nil {
							return err
						}
						*h.Password = v
					}

					if h.Password != nil && strings.ContainsRune(*h.Password, '$') {
						v, err := env.Expand(*h.Password, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
						if err != nil {
							return err
						}
						*h.Password = v
					}

					if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '{') {
						v, err := eval.EvalAsString(*h.IdentityFile, scope)
						if err != nil {
							return err
						}
						*h.IdentityFile = v
					}

					if h.IdentityFile != nil && strings.ContainsRune(*h.IdentityFile, '$') {
						v, err := env.Expand(*h.IdentityFile, env.WithGet(p.Env.Get), env.WithCommandSubstitution(substitution))
						if err != nil {
							return err
						}
						*h.IdentityFile = v
					}

					for _, h2 := range p.Hosts {
						if h2.Host == h.Host {
							return errors.Newf("duplicate host entry for host %s", h.Host)
						}
					}

					port := uint(22)
					if h.Port != nil && *h.Port > 0 {
						port = *h.Port
					}

					user := ""
					if h.User != nil {
						user = *h.User
					}

					password := ""
					if h.Password != nil {
						password = *h.Password
					}

					identityFile := ""
					if h.IdentityFile != nil {
						identityFile = *h.IdentityFile
					}

					osInfo := types.OsInfo{}
					if h.OS != nil {
						osInfo = *h.OS
					}

					meta := types.NewMeta()
					if h.Meta != nil {
						meta = h.Meta
					}

					p.Hosts[hostName] = HostInfo{
						Host:         h.Host,
						Port:         port,
						User:         user,
						Password:     password,
						IdentityFile: identityFile,
						OS:           osInfo,
						Meta:         *meta,
						Tags:         h.Tags,
					}

					if _, ok := p.Hosts[h.Host]; !ok {
						p.Hosts[h.Host] = p.Hosts[hostName]
					}
				}
			}

			ns := mod.Namespace

			if mod.Tasks != nil && mod.Tasks.Len() > 0 && len(mod.TaskNames) == 0 {
				mod.TaskNames = mod.Tasks.Keys()
			}

			if len(mod.TaskNames) > 0 {
				for _, taskName := range mod.TaskNames {
					task, ok := mod.Tasks.Get(taskName)
					if !ok {
						return errors.Newf("module %s does not have task %s", path, taskName)
					}

					name := task.Name
					if ns != "" {
						name = ns + ":" + name
						task.Name = name
						if task.Id != "" {
							task.Id = ns + "-" + task.Id
						}
					}

					if task.Id == "" {
						task.Id = id.Convert(name)
					}

					task.Env.Set("CAST_FILE", mod.File)
					task.Env.Set("CAST_DIR", mod.Dir)
					task.Env.Set("CAST_PARENT_FILE", p.File)
					task.Env.Set("CAST_PARENT_DIR", p.Dir)
					task.Env.Set("CAST_MODULE_ID", mod.Id)
					task.Env.Set("CAST_MODULE_NAME", mod.Name)
					task.Env.Set("CAST_MODULE_VERSION", mod.Version)
					task.Env.Set("CAST_MODULE_DESCRIPTION", mod.Desc)
					p.Tasks.Set(&task)
				}
			} else {

				for _, task := range mod.Tasks.Values() {
					name := task.Name
					if ns != "" {
						name = ns + ":" + name
						task.Name = name
						if task.Id != "" {
							task.Id = ns + "-" + task.Id
						}
					}

					if task.Id == "" {
						task.Id = id.Convert(name)
					}

					task.Env.Set("CAST_FILE", mod.File)
					task.Env.Set("CAST_DIR", mod.Dir)
					task.Env.Set("CAST_PARENT_FILE", p.File)
					task.Env.Set("CAST_PARENT_DIR", p.Dir)
					task.Env.Set("CAST_MODULE_ID", mod.Id)
					task.Env.Set("CAST_MODULE_NAME", mod.Name)
					task.Env.Set("CAST_MODULE_VERSION", mod.Version)
					task.Env.Set("CAST_MODULE_DESCRIPTION", mod.Desc)
					p.Tasks.Set(&task)
				}
			}
		}
	}

	return nil
}

func resolveModules(p *Project, imports *types.Imports) error {

	if imports == nil || len(*imports) == 0 {
		return nil
	}

	dataDir, err := paths.UserDataDir()
	if err != nil {
		return err
	}

	castDir := p.CastDir

	moduleDir := ""
	if castDir != dataDir {
		moduleDir = filepath.Join(castDir, "modules")
	}

	usersModuleDir := filepath.Join(dataDir, "modules")

	for _, importMod := range *imports {
		path := importMod.From
		if !filepath.IsAbs(path) {
			if path[0] == '.' {
				absPath, err := filepath.Abs(filepath.Join(p.Dir, path))
				if err != nil {
					return err
				}
				path = absPath
			} else {
				if moduleDir != "" {
					path = filepath.Join(moduleDir, path)
					if _, err := os.Stat(path); err != nil {
						path = filepath.Join(usersModuleDir, importMod.From)
					}
				} else {
					path = filepath.Join(usersModuleDir, importMod.From)
				}
			}
		}

		if _, err := os.Stat(path); err != nil {
			return err
		}

		// already imported
		if _, exists := p.imported[path]; exists {
			continue
		}

		mod := types.Module{}
		err := mod.ReadFromYaml(path)
		if err != nil {
			return err
		}

		if mod.Imports != nil && len(*mod.Imports) > 0 {
			err := resolveModules(p, mod.Imports)
			if err != nil {
				return err
			}
		}

		mod.Namespace = importMod.Namespace
		mod.TaskNames = importMod.Tasks
		p.imported[path] = mod
		p.importedOrder = append(p.importedOrder, path)
	}
	return nil

}

func setupEnv(p *Project) error {

	e := types.NewEnv()
	e.Merge(globalEnv)

	sub := true
	if p.Schema.Config != nil && p.Schema.Config.Substitution != nil {
		sub = *p.Schema.Config.Substitution
	}

	for _, path := range p.importedOrder {
		mod := p.imported[path]
		if mod.Paths == nil {
			continue
		}
		err := loadPaths(*mod.Paths, e, mod.Dir)
		if err != nil {
			return err
		}
	}

	if p.Schema.Paths != nil {
		loadPaths(*p.Schema.Paths, e, p.Dir)
	}

	f := e.Get("CAST_PATH")
	if f != "" {
		if !filepath.IsAbs(f) {
			f = filepath.Join(p.Dir, f)
		}

		if _, err := os.Stat(f); err == nil {
			content, err := os.ReadFile(f)
			if err != nil {
				return err
			}

			paths := strings.Split(string(content), "\n")
			for _, p := range paths {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				err := e.PrependPath(p)
				if err != nil {
					return err
				}
			}
		}

	} else {
		f, err := os.CreateTemp("", "cast-path-")
		if err != nil {
			return err
		}
		f.Write([]byte{})
		f.Close()
		e.Set("CAST_PATH", f.Name())
		p.cleanupPath = true
	}

	e.PrependPath("./bin")
	e.PrependPath("./node_modules/.bin")

	for _, path := range p.importedOrder {
		mod := p.imported[path]
		if mod.DotEnv == nil {
			continue
		}

		_, err := loadDotEnvFiles(*mod.DotEnv, e, p.ContextName, sub, mod.Dir)
		if err != nil {
			return err
		}
	}

	if p.Schema.DotEnv != nil {
		_, err := loadDotEnvFiles(*p.Schema.DotEnv, e, p.ContextName, sub, p.Dir)
		if err != nil {
			return err
		}
	}

	for _, path := range p.importedOrder {
		mod := p.imported[path]
		err := loadEnv(mod.Env, e, sub)
		if err != nil {
			return err
		}
	}

	if p.Schema.Env != nil {
		err := loadEnv(p.Schema.Env, e, sub)
		if err != nil {
			return err
		}
	}

	f = e.Get("CAST_ENV")
	if f != "" {
		envFile := f
		if !filepath.IsAbs(envFile) {
			envFile = filepath.Join(p.Dir, envFile)
		}
		if _, err := os.Stat(envFile); err == nil {

			bytes, err := os.ReadFile(envFile)
			if err != nil {
				return err
			}

			dotenvDoc, err := dotenv.Parse(string(bytes))
			if err != nil {
				return err
			}

			opts := &env.ExpandOptions{
				Get: e.Get,
				Set: func(s1, s2 string) error {
					e.Set(s1, s2)
					return nil
				},
				Keys:                e.Keys(),
				CommandSubstitution: sub,
			}

			for _, node := range dotenvDoc.ToArray() {
				if node.Type != dotenv.VARIABLE {
					continue
				}

				key := node.Key
				if key == nil || len(*key) == 0 {
					continue
				}

				value := node.Value

				v, err := env.ExpandWithOptions(value, opts)
				if err != nil {
					return err
				}

				e.Set(*key, v)
			}
		}

	} else {
		f, err := os.CreateTemp("", "cast-env-")
		if err != nil {
			return err
		}
		f.Write([]byte{})
		f.Close()
		e.Set("CAST_ENV", f.Name())
		p.cleanupEnv = true
	}

	f = e.Get("CAST_OUTPUTS")
	if f == "" {
		f, err := os.CreateTemp("", "cast-outputs-")
		if err != nil {
			return err
		}
		f.Write([]byte{})
		f.Close()
		e.Set("CAST_OUTPUTS", f.Name())
		p.cleanupOutputs = true
	}

	p.Env = e

	return nil
}

func loadPaths(src types.Paths, dest *types.Env, basePath string) error {

	for _, p := range src {
		if p.OS != "" && p.OS != "*" {
			if p.OS != runtime.GOOS {
				continue
			}
		}

		resolvedPath, err := paths.ResolvePath(basePath, p.Path)
		if err != nil {
			return err
		}

		if p.Append {
			err := dest.AppendPath(resolvedPath)
			if err != nil {
				return err
			}

			continue
		}

		err = dest.PrependPath(resolvedPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadEnv(src *types.Env, dest *types.Env, substitution bool) error {
	opts := &env.ExpandOptions{
		Get: dest.Get,
		Set: func(s1, s2 string) error {
			dest.Set(s1, s2)
			return nil
		},
		Keys:                dest.Keys(),
		CommandSubstitution: substitution,
	}

	for _, key := range src.Keys() {
		value := src.Get(key)
		v, err := env.ExpandWithOptions(value, opts)
		if err != nil {
			return err
		}

		dest.Set(key, v)
	}

	return nil
}

func loadDotEnvFiles(dotenvSection types.DotEnvs, e *types.Env, contextName string, subsitution bool, basePath string) (*types.Env, error) {
	dotenvFiles := []string{}
	for _, section := range dotenvSection {
		if section.OS != "" && section.OS != "*" {
			if section.OS != runtime.GOOS {
				continue
			}
		}

		if len(section.Contexts) == 0 && (contextName == "" || contextName == "*" || contextName == "default") {
			dotenvFiles = append(dotenvFiles, section.Path)
			continue
		}

		for _, ctx := range section.Contexts {
			if ctx == contextName || ctx == "*" {
				dotenvFiles = append(dotenvFiles, section.Path)
				break
			}
		}
	}

	opts := &env.ExpandOptions{
		Get: e.Get,
		Set: func(s1, s2 string) error {
			e.Set(s1, s2)
			return nil
		},
		Keys:                e.Keys(),
		CommandSubstitution: subsitution,
	}

	globalDoc := dotenv.NewDoc()

	for _, file := range dotenvFiles {
		absFile, err := paths.ResolvePath(basePath, file)
		println("dotenv file", absFile)
		if err != nil {
			return nil, err
		}

		file = absFile
		if strings.HasSuffix(file, "?") {
			if _, err := os.Stat(file[:len(file)-1]); err != nil {
				continue
			}

			file = file[:len(file)-1]
		} else {
			if _, err := os.Stat(file); err != nil {
				return nil, err
			}
		}

		bytes, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}

		nextDoc, err := dotenv.Parse(string(bytes))
		if err != nil {
			return nil, err
		}

		globalDoc.Merge(nextDoc)
	}

	for _, node := range globalDoc.ToArray() {
		if node.Type != dotenv.VARIABLE {
			continue
		}

		key := node.Key
		if key == nil || len(*key) == 0 {
			continue
		}

		value := node.Value

		v, err := env.ExpandWithOptions(value, opts)
		if err != nil {
			return nil, err
		}

		e.Set(*key, v)
	}

	return e, nil
}
