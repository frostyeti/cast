package models

type Step struct {
	Id        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Uses      string `json:"uses,omitempty"`
	Run       string `json:"run,omitempty"`
	Inputs    Inputs `json:"with,omitempty"`
	Env       Env    `json:"env,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
	Desc      string `json:"desc,omitempty"`
	Force     bool   `json:"force,omitempty"`
	Predicate bool   `json:"predicate,omitempty"`
	Extends   string `json:"extends,omitempty"`
}
