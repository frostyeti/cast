package types

import (
	"iter"
	"maps"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
	"go.yaml.in/yaml/v4"
)

type Env struct {
	Map  map[string]string
	keys []string
}

func (e *Env) MarshalYAML() (interface{}, error) {
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

func (e *Env) UnmarshalYAML(node *yaml.Node) error {
	if e == nil {
		e = &Env{}
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

func NewEnv() *Env {
	return &Env{
		Map:  map[string]string{},
		keys: []string{},
	}
}

func (e *Env) Set(key, value string) {
	e.init()

	if _, ok := e.Map[key]; !ok {
		e.keys = append(e.keys, key)
	}

	e.Map[key] = value
}

func (e *Env) Get(key string) string {
	e.init()
	val, _ := e.Map[key]
	return val
}

func (e *Env) Has(key string) bool {
	if e == nil {
		e = NewEnv()
	}
	_, ok := e.Map[key]
	return ok
}

func (e *Env) TryGet(key string) (string, bool) {
	if e == nil {
		e = NewEnv()
	}
	val, ok := e.Map[key]
	return val, ok
}

func (e *Env) Clone() *Env {
	if e == nil {
		return NewEnv()
	}
	clone := NewEnv()

	for k, v := range e.Map {
		clone.Map[k] = v
	}
	clone.keys = append(clone.keys, e.keys...)
	return clone
}

func (e *Env) ToMap() map[string]string {
	if e == nil {
		return map[string]string{}
	}
	m := make(map[string]string, len(e.Map))
	maps.Copy(m, e.Map)
	return m
}

func (e *Env) Keys() []string {
	if e == nil {
		return []string{}
	}
	keys := slices.Clone(e.keys)
	return keys
}

func (e *Env) Values() []string {
	e.init()
	values := make([]string, 0, len(e.Map))
	for _, k := range e.keys {
		values = append(values, e.Map[k])
	}
	return values
}

func (e *Env) Len() int {
	if e == nil {
		return 0
	}

	e.init()
	return len(e.Map)
}

func (e *Env) init() {
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

func (e *Env) Merge(other *Env) {
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

// create iterator over Env that requires iter.Seq[string, string]

func (e *Env) Iter() iter.Seq2[string, string] {
	e.init()
	return func(yield func(string, string) bool) {
		for _, k := range e.keys {
			if !yield(k, e.Map[k]) {
				return
			}
		}
	}
}

func (e *Env) PrependPath(path string) error {
	e.init()
	paths := e.SplitPath()

	if len(paths) > 0 {
		if runtime.GOOS == "windows" {
			if strings.EqualFold(paths[0], path) {
				return nil
			}
		} else {
			if paths[0] == path {
				return nil
			}
		}
	}

	paths = append([]string{path}, paths...)
	e.SetPath(strings.Join(paths, string(os.PathListSeparator)))
	return nil
}

func (e *Env) AppendPath(path string) error {
	e.init()
	paths := e.SplitPath()

	if len(paths) > 0 {
		if runtime.GOOS == "windows" {
			for _, p := range paths {
				if strings.EqualFold(p, path) {
					return nil
				}
			}
		} else {
			for _, p := range paths {
				if p == path {
					return nil
				}
			}
		}
	}

	paths = append(paths, path)
	e.SetPath(strings.Join(paths, string(os.PathListSeparator)))
	return nil
}

func (e *Env) HasPath(path string) bool {
	e.init()
	paths := e.SplitPath()
	if runtime.GOOS == "windows" {
		for _, p := range paths {
			if strings.EqualFold(p, path) {
				return true
			}
		}
		return false
	}

	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

func (e *Env) SplitPath() []string {
	e.init()
	if e.GetPath() == "" {
		return []string{}
	}
	return strings.Split(e.GetPath(), string(os.PathListSeparator))
}

func (e *Env) GetPath() string {
	e.init()
	if runtime.GOOS == "windows" {
		if val, ok := e.Map["Path"]; ok {
			return val
		}

		return ""
	}

	if val, ok := e.Map["PATH"]; ok {
		return val
	}

	return ""
}

func (e *Env) SetPath(value string) error {
	e.init()
	if runtime.GOOS == "windows" {
		e.Map["Path"] = value
		return nil
	}

	e.Map["PATH"] = value
	return nil
}
