package schemas

type ModuleConfig struct {
	Id       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Desc     string   `yaml:"description"`
	License  *string  `yaml:"license,omitempty"`
	Authors  []string `yaml:"authors,omitempty"`
	Version  string   `yaml:"version"`
	Handlers []string `yaml:"handlers,omitempty"`
	Env      *Env     `yaml:"env,omitempty"`
	Tasks    TaskMap  `yaml:"tasks,omitempty"`
	Needs    []Need   `yaml:"needs,omitempty"`
}
