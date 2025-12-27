package models

import "github.com/frostyeti/cast/internal/eval"

type Scope struct {
	Env       map[string]string
	OS        map[string]interface{}
	Outputs   map[string]interface{}
	Git       map[string]string
	Inputs    map[string]interface{}
	FrostYeti map[string]interface{}
}

func (s *Scope) ToModel() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range s.Env {
		result[k] = v
	}

	result["os"] = s.OS
	result["outputs"] = s.Outputs
	result["git"] = s.Git
	result["inputs"] = s.Inputs
	result["fy"] = s.FrostYeti

	return result
}

func (s *Scope) Eval(template string) (string, error) {
	return eval.EvalTemplate(template, s.ToModel())
}
