package schemas

type Import struct {
	From      string   `yaml:"from,omitempty"`
	Namespace string   `yaml:"namespace,omitempty"`
	Tasks     []string `yaml:"tasks,omitempty"`
}
