package schemas

type WorkspaceConfig struct {
	Projects *Projects `yaml:"projects,omitempty"`
	Defaults *Defaults `yaml:"defaults,omitempty"`
	Env      *Env      `yaml:"env,omitempty"`
	Modules  []string  `yaml:"modules,omitempty"`
}
