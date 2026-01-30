package types

type Step struct {
	Id       *string `json:"id,omitempty"`
	Name     *string `json:"name,omitempty"`
	Uses     string  `json:"uses,omitempty"`
	Run      string  `json:"run,omitempty"`
	With     With    `json:"with,omitempty"`
	Env      Env     `json:"env,omitempty"`
	Cwd      *string `json:"cwd,omitempty"`
	Desc     *string `json:"desc,omitempty"`
	Force    *string `json:"force,omitempty"`
	TaskName *string `json:"task,omitempty"`
}

type Steps []Step
