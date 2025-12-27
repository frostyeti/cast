package schemas

type Projects struct {
	Include []string          `yaml:"include,omitempty"`
	Exclude []string          `yaml:"exclude,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
}
