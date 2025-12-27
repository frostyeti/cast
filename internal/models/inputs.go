package models

import "github.com/frostyeti/cast/internal/eval"

type Input struct {
	evaluated   bool
	value       any
	Id          string
	RawValue    any
	Value       any
	Description string
	Type        string
	Default     any
	Required    bool
}

type Scope struct {
	Env     map[string]string
	OS      map[string]interface{}
	Outputs map[string]interface{}
	Git     map[string]string
	Inputs  map[string]interface{}
}

func (s *Scope) ToModel() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range s.Env {
		result[k] = v
	}

	result["os"] = s.OS
	result["outputs"] = s.Outputs
	result["git"] = s.Git

	return result
}

func (i *Input) Eval() any {
	if i.evaluated {
		return i.value
	}

	if i.RawValue == nil && i.Default != nil {
		i.value = i.Default
		i.evaluated = true
		return i.value
	}

	if strVal, ok := i.RawValue.(string); ok {
		evaled, err := eval.EvalTemplate(strVal, map[string]interface{}{})
		if err == nil {
			i.value = evaled
			i.evaluated = true
			return i.value
		}
	}

	i.value = i.RawValue
	i.evaluated = true
	return i.value
}

type Inputs struct {
	Map  map[string]Input
	keys []string
}

func NewInputs() Inputs {
	return Inputs{
		Map:  make(map[string]Input),
		keys: []string{},
	}
}

func (i *Inputs) Set(key string, input Input) {
	if _, exists := i.Map[key]; !exists {
		i.keys = append(i.keys, key)
	}
	i.Map[key] = input
}

func (e *Inputs) Keys() []string {
	return e.keys
}

func (e *Inputs) Values() []Input {
	values := make([]Input, 0, len(e.keys))
	for _, key := range e.keys {
		values = append(values, e.Map[key])
	}
	return values
}

func (e *Inputs) Get(key string) (Input, bool) {
	value, exists := e.Map[key]
	return value, exists
}
