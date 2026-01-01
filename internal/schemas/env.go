package schemas

import (
	"maps"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Env struct {
	Files   []DotEnvFile  `yaml:"files,omitempty"`
	Paths   []PrependPath `yaml:"paths,omitempty"`
	Vars    EnvVars       `yaml:"vars,omitempty"`
	Secrets []string      `yaml:"secrets,omitempty"`
}

func (e *Env) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})

	if len(e.Files) > 0 {
		files, err := e.MarshalYAML()
		if err != nil {
			return nil, err
		}

		mapping["files"] = files
	}

	if len(e.Paths) > 0 {
		paths, err := e.MarshalYAML()
		if err != nil {
			return nil, err
		}

		mapping["paths"] = paths
	}

	if e.Vars.Len() > 0 {
		vars, err := e.Vars.MarshalYAML()
		if err != nil {
			return nil, err
		}

		mapping["vars"] = vars
	}

	if len(e.Secrets) > 0 {
		mapping["secrets"] = e.Secrets
	}

	return mapping, nil
}

func (e *Env) UnmarshalYAML(node *yaml.Node) error {
	if e == nil {
		e = &Env{}
	}

	if node.Kind != yaml.MappingNode {
		return errors.YamlErrorf(node, "expected yaml mapping for env")
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		switch keyNode.Value {
		case "files":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.YamlErrorf(valueNode, "expected yaml sequence for 'files' field")
			}
			for _, item := range valueNode.Content {
				next := DotEnvFile{
					Contexts: []string{"*"},
					OS:       "",
				}

				if item.Kind == yaml.ScalarNode {
					next.Path = item.Value
					continue
				}

				if item.Kind != yaml.MappingNode {
					return errors.YamlErrorf(item, "expected yaml mapping or scalar for 'files' item")
				}

				keyNode := item.Content[0]
				valueNode := item.Content[1]

				switch keyNode.Value {
				case "windows", "win", "win32":
					next.OS = "windows"
					next.Path = valueNode.Value
				case "linux":
					next.OS = "linux"
					next.Path = valueNode.Value
				case "darwin", "mac", "macos":
					next.OS = "darwin"
					next.Path = valueNode.Value
				case "path":
					next.Path = valueNode.Value
				case "contexts":
					if valueNode.Kind == yaml.ScalarNode {
						next.Contexts = []string{valueNode.Value}
						continue
					}

					if valueNode.Kind != yaml.SequenceNode {
						return errors.YamlErrorf(valueNode, "expected yaml sequence for 'contexts' field in 'files' item")
					}

					for _, ctxItem := range valueNode.Content {
						if ctxItem.Kind != yaml.ScalarNode {
							return errors.YamlErrorf(ctxItem, "expected yaml scalar in 'contexts' sequence in 'files' item")
						}
						next.Contexts = append(next.Contexts, ctxItem.Value)
					}
				default:
					return errors.YamlErrorf(keyNode, "unexpected field '%s' in 'files' item", keyNode.Value)
				}
				e.Files = append(e.Files, next)

			}
		case "paths":
			if valueNode.Kind != yaml.SequenceNode {
				return errors.YamlErrorf(valueNode, "expected yaml sequence for 'paths' field")
			}
			for _, item := range valueNode.Content {
				next := PrependPath{
					OS:     "",
					Append: false,
				}

				if item.Kind == yaml.ScalarNode {
					next.Value = item.Value
					e.Paths = append(e.Paths, next)
					continue
				}

				if item.Kind != yaml.MappingNode {
					return errors.YamlErrorf(item, "expected yaml mapping or scalar for 'paths' item")
				}

				for j := 0; j < len(item.Content); j += 2 {
					keyNode := item.Content[j]
					valueNode := item.Content[j+1]

					switch keyNode.Value {
					case "windows", "win", "win32":
						next.OS = "windows"
						next.Value = valueNode.Value
					case "linux":
						next.OS = "linux"
						next.Value = valueNode.Value
					case "darwin", "mac", "macos":
						next.OS = "darwin"
						next.Value = valueNode.Value
					case "value", "path":
						if valueNode.Kind != yaml.ScalarNode {
							return errors.YamlErrorf(valueNode, "expected yaml scalar for 'value' field in 'paths' item")
						}
						next.Value = valueNode.Value
					case "os":
						if valueNode.Kind != yaml.ScalarNode {
							return errors.YamlErrorf(valueNode, "expected yaml scalar for 'os' field in 'paths' item")
						}
						next.OS = valueNode.Value
					case "append":
						if valueNode.Kind != yaml.ScalarNode {
							return errors.YamlErrorf(valueNode, "expected yaml scalar for 'append' field in 'paths' item")
						}
						if valueNode.Value == "true" || valueNode.Value == "1" {
							next.Append = true
						} else {
							next.Append = false
						}
					default:
						return errors.YamlErrorf(keyNode, "unexpected field '%s' in 'paths' item", keyNode.Value)
					}
				}
				e.Paths = append(e.Paths, next)
			}
		case "vars":
			var envVars EnvVars
			if err := valueNode.Decode(&envVars); err != nil {
				return err
			}
			e.Vars = envVars
		case "secrets":
			if valueNode.Kind == yaml.ScalarNode {
				e.Secrets = []string{valueNode.Value}
				continue
			}

			if valueNode.Kind != yaml.SequenceNode {
				return errors.YamlErrorf(valueNode, "expected yaml sequence for 'secrets' field")
			}

			for _, item := range valueNode.Content {
				if item.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(item, "expected yaml scalar in 'secrets' sequence")
				}
				e.Secrets = append(e.Secrets, item.Value)
			}
		}
	}

	return nil
}

type PrependPath struct {
	Value  string `json:"value"`
	OS     string `json:"os,omitempty"`
	Append bool   `json:"append,omitempty"`
}

func (pp *PrependPath) MarshalYAML() (interface{}, error) {
	if pp.OS == "" && !pp.Append {
		return pp.Value, nil
	}

	mapping := make(map[string]interface{})
	if pp.Append == false && pp.OS == "" || pp.OS == "*" {
		mapping[pp.OS] = pp.Value
		return mapping, nil
	}

	mapping["path"] = pp.Value

	if pp.OS != "" && pp.OS != "*" {
		mapping["os"] = pp.OS
	}

	if pp.Append {
		mapping["append"] = pp.Append
	}

	return mapping, nil
}

type DotEnvFile struct {
	Path     string   `yaml:"path"`
	OS       string   `yaml:"os,omitempty"`
	Contexts []string `yaml:"contexts,omitempty"`
}

func (df *DotEnvFile) HasContext(context string) bool {
	if len(df.Contexts) == 0 {
		return context == "*" || context == "" || context == "default"
	}

	for _, ctx := range df.Contexts {
		if ctx == "*" || ctx == context {
			return true
		}
	}

	return false
}

func (df *DotEnvFile) MarshalYAML() (interface{}, error) {
	l := len(df.Contexts)
	isDefaultContext := l == 0 || (l == 1 && df.Contexts[0] == "*")

	if df.OS == "" && isDefaultContext {
		return df.Path, nil
	}

	mapping := make(map[string]interface{})

	if df.OS != "" && df.OS != "*" && isDefaultContext {
		mapping[df.OS] = df.Path
		return mapping, nil
	}

	mapping["path"] = df.Path

	if df.OS != "" && df.OS != "*" {
		mapping["os"] = df.OS
	}

	if !isDefaultContext {
		mapping["contexts"] = df.Contexts
	}

	return mapping, nil
}

type EnvVars struct {
	Map  map[string]string
	keys []string
}

func (e *EnvVars) MarshalYAML() (interface{}, error) {
	mapping := make(map[string]interface{})
	for _, k := range e.keys {
		mapping[k] = e.Map[k]
	}
	return mapping, nil
}

type EnvVarsVariable struct {
	Name     string
	Value    string
	File     string
	IsSecret bool
}

func (ev *EnvVarsVariable) UnmarshalYAML(node *yaml.Node) error {
	if ev == nil {
		ev = &EnvVarsVariable{}
	}

	if node.Kind == yaml.ScalarNode {
		if strings.ContainsRune(node.Value, '=') {
			parts := strings.SplitN(node.Value, "=", 2)
			ev.Name = parts[0]
			ev.Value = parts[1]
			ev.IsSecret = false
			return nil
		} else if strings.ContainsRune(node.Value, ':') {
			parts := strings.SplitN(node.Value, ":", 2)
			ev.Name = parts[0]
			ev.Value = parts[1]
			ev.IsSecret = true
			return nil
		} else {
			return errors.YamlErrorf(node, "invalid EnvVars variable format, expected 'KEY=VALUE' or 'KEY:VALUE'")
		}
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			key := keyNode.Value
			switch key {
			case "name":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'name' field")
				}
				ev.Name = valueNode.Value
			case "value":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'value' field")
				}
				ev.Value = valueNode.Value
				ev.IsSecret = false
			case "secret":
				if valueNode.Kind != yaml.ScalarNode {
					return errors.YamlErrorf(valueNode, "expected yaml scalar for 'secret' field")
				}
				if valueNode.Value == "true" || valueNode.Value == "1" {
					ev.IsSecret = true
				} else {
					ev.IsSecret = false
				}
			default:
				return errors.YamlErrorf(keyNode, "unexpected field '%s' in EnvVars variable", key)
			}
		}

		return nil
	}

	return errors.YamlErrorf(node, "expected yaml scalar or mapping for EnvVars variable")
}

func (e *EnvVars) UnmarshalYAML(node *yaml.Node) error {
	if e == nil {
		e = &EnvVars{}
	}

	e.Map = make(map[string]string)
	e.keys = []string{}

	if node.Kind == yaml.SequenceNode {
		for _, itemNode := range node.Content {
			var ev EnvVarsVariable
			if err := itemNode.Decode(&ev); err != nil {
				return err
			}

			var name = ev.Name
			if name == "" {
				return errors.YamlErrorf(itemNode, "EnvVars variable name cannot be empty")
			}

			e.Map[ev.Name] = ev.Value
			hasKey := false
			for _, k := range e.keys {
				if k == ev.Name {
					hasKey = true
					break
				}
			}
			if !hasKey {
				e.keys = append(e.keys, ev.Name)
			}
		}
		return nil
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			hasKey := false

			if keyNode.Kind != yaml.ScalarNode {
				return errors.YamlErrorf(keyNode, "expected yaml scalar for EnvVars variable name")
			}
			name := keyNode.Value
			if name == "" {
				return errors.YamlErrorf(keyNode, "EnvVars variable name cannot be empty")
			}

			if valueNode.Kind == yaml.ScalarNode {
				e.Map[name] = valueNode.Value

				for _, k := range e.keys {
					if k == name {
						hasKey = true
						break
					}
				}
				if !hasKey {
					e.keys = append(e.keys, name)
				}

				continue
			}

			if valueNode.Kind == yaml.MappingNode {
				ev := &EnvVarsVariable{Name: name}
				if err := valueNode.Decode(ev); err != nil {
					return err
				}

				ev.Name = name
				e.Map[ev.Name] = ev.Value

				for _, k := range e.keys {
					if k == ev.Name {
						hasKey = true
						break
					}
				}

				if !hasKey {
					e.keys = append(e.keys, ev.Name)
				}

				continue
			}

			return errors.YamlErrorf(valueNode, "expected yaml scalar or mapping for EnvVars variable value")
		}

		return nil
	}

	return errors.YamlErrorf(node, "expected yaml sequence for EnvVars")
}

func NewEnv() *EnvVars {
	return &EnvVars{
		Map:  map[string]string{},
		keys: []string{},
	}
}

func (e *EnvVars) Set(key, value string) {
	e.init()

	if _, ok := e.Map[key]; !ok {
		e.keys = append(e.keys, key)
	}

	e.Map[key] = value
}

func (e *EnvVars) Get(key string) (string, bool) {
	e.init()
	val, ok := e.Map[key]
	return val, ok
}

func (e *EnvVars) Clone() *EnvVars {
	e.init()
	clone := NewEnv()

	for k, v := range e.Map {
		clone.Map[k] = v
	}
	clone.keys = append(clone.keys, e.keys...)
	return clone
}

func (e *EnvVars) ToMap() map[string]string {
	e.init()
	m := make(map[string]string, len(e.Map))
	maps.Copy(m, e.Map)
	return m
}

func (e *EnvVars) Keys() []string {
	e.init()
	keys := make([]string, 0, len(e.Map))
	for k := range e.Map {
		keys = append(keys, k)
	}
	return keys
}

func (e *EnvVars) Values() []string {
	e.init()
	values := make([]string, 0, len(e.Map))
	for _, k := range e.keys {
		values = append(values, e.Map[k])
	}
	return values
}

func (e *EnvVars) Len() int {
	if e == nil {
		return 0
	}

	e.init()
	return len(e.Map)
}

func (e *EnvVars) init() {
	if e == nil {
		e = NewEnv()
	}

	if e.Map == nil {
		e.Map = map[string]string{}
	}

	if e.keys == nil {
		e.keys = []string{}
	}

}

func (e *EnvVars) Merge(other *EnvVars) {
	e.init()
	if other == nil {
		return
	}
	other.init()

	for _, k := range other.keys {
		e.Map[k] = other.Map[k]

		hasKey := false
		for _, ek := range e.keys {
			if ek == k {
				hasKey = true
				break
			}
		}
		if !hasKey {
			e.keys = append(e.keys, k)
		}
	}

}
