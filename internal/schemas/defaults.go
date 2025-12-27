package schemas

type Defaults struct {
	Context *string        `yaml:"context,omitempty"`
	Shell   *string        `yaml:"shell,omitempty"`
	Cache   *CacheDefaults `yaml:"cache,omitempty"`
	Remote  *bool          `yaml:"remote,omitempty"`
}

type CacheDefaults struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}
