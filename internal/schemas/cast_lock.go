package schemas

type CastLock struct {
	Version string           `yaml:"version"`
	Modules []CastLockModule `yaml:"modules"`
}

type CastLockModule struct {
	Id       string `yaml:"id"`
	Version  string `yaml:"version"`
	Checksum string `yaml:"checksum"`
	Url      string `yaml:"url"`
	Needs    []Need `yaml:"needs,omitempty"`
}
