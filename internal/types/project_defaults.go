package types

// ProjectDefaults holds project-wide defaults.
type ProjectDefaults struct {
	Shell *string `yaml:"shell,omitempty" json:"shell,omitempty"`
}
