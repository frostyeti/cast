package schemas

type CastConfig struct {
	Imports  []Import  `yaml:"imports,omitempty"`
	Id       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Desc     *string   `yaml:"description,omitempty"`
	Version  *string   `yaml:"version,omitempty"`
	Defaults *Defaults `yaml:"defaults,omitempty"`
	Env      *Env      `yaml:"env,omitempty"`
	Tasks    TaskMap   `yaml:"tasks,omitempty"`
	Needs    []Need    `yaml:"needs,omitempty"`
}
