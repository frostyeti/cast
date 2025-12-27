package models

import (
	"iter"
	"maps"
	"os"
	"runtime"
	"strings"

	"github.com/frostyeti/go/env"
)

type Env struct {
	Items map[string]string

	keys []string
}

func NewEnv() *Env {
	return &Env{
		Items: make(map[string]string),
		keys:  []string{},
	}
}

func NewEnvFromEnviron() *Env {
	env := NewEnv()
	for _, envVar := range os.Environ() {
		// Split the environment variable into key and value
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			env.Set(parts[0], parts[1])
		}
	}
	return env
}

func (e *Env) Set(key, value string) {
	e.init()

	if _, ok := e.Items[key]; !ok {
		e.keys = append(e.keys, key)
	}

	e.Items[key] = value
}

func (e *Env) Get(key string) string {
	e.init()
	val, _ := e.Items[key]
	return val
}

func (e *Env) TryGet(key string) (string, bool) {
	e.init()
	val, ok := e.Items[key]
	return val, ok
}

func (e *Env) Has(key string) bool {
	e.init()
	_, ok := e.Items[key]
	return ok
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
		if val, ok := e.Items["Path"]; ok {
			return val
		}

		return ""
	}

	if val, ok := e.Items["PATH"]; ok {
		return val
	}

	return ""
}

func (e *Env) SetPath(value string) error {
	e.init()
	if runtime.GOOS == "windows" {
		e.Items["Path"] = value
		return nil
	}

	e.Items["PATH"] = value
	return nil
}

func (e *Env) GetString(key string) string {
	e.init()
	if val, ok := e.Items[key]; ok {
		return val
	}
	return ""
}

func (e *Env) Delete(key string) {
	e.init()
	delete(e.Items, key)
	for i, k := range e.keys {
		if k == key {
			e.keys = append(e.keys[:i], e.keys[i+1:]...)
			break
		}
	}
}

func (e *Env) Clone() *Env {
	e.init()
	clone := NewEnv()

	for k, v := range e.Items {
		clone.Items[k] = v
	}
	clone.keys = append(clone.keys, e.keys...)
	return clone
}

func (e *Env) ToMap() map[string]string {
	e.init()
	m := make(map[string]string, len(e.Items))
	maps.Copy(m, e.Items)
	return m
}

func (e *Env) Keys() []string {
	e.init()
	keys := make([]string, 0, len(e.Items))
	for k := range e.Items {
		keys = append(keys, k)
	}
	return keys
}

func (e *Env) Values() []string {
	e.init()
	values := make([]string, 0, len(e.Items))
	for _, k := range e.keys {
		values = append(values, e.Items[k])
	}
	return values
}

func (e *Env) Len() int {
	if e == nil {
		return 0
	}

	e.init()
	return len(e.Items)
}

// return iter.Seq
func (e *Env) Iter() iter.Seq2[string, string] {
	e.init()
	return func(yield func(string, string) bool) {
		for _, k := range e.keys {
			if !yield(k, e.Items[k]) {
				break
			}
		}
	}
}

func (e *Env) init() {
	if e == nil {
		e = NewEnv()
	}

	if e.Items == nil {
		e.Items = map[string]string{}
	}

	if e.keys == nil {
		e.keys = []string{}
	}

}

func (e *Env) Merge(other *Env) {
	if e == nil {
		e.init()
	}

	if other == nil {
		return
	}

	for _, k := range other.keys {
		e.Items[k] = other.Items[k]

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

func (e *Env) Expand(s string) (string, error) {
	if e == nil {
		return s, nil
	}
	opts := env.ExpandOptions{
		Get: e.GetString,
		Set: func(key, value string) error {
			e.Set(key, value)
			return nil
		},
		Keys:                e.Keys(),
		CommandSubstitution: true,
	}

	return env.ExpandWithOptions(s, &opts)
}
