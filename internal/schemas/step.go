package schemas

type Step struct {
	Id        *string `json:"id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Uses      string  `json:"uses,omitempty"`
	Run       string  `json:"run,omitempty"`
	With      With    `json:"with,omitempty"`
	Env       EnvVars `json:"env,omitempty"`
	Cwd       *string `json:"cwd,omitempty"`
	Desc      *string `json:"desc,omitempty"`
	Force     *string `json:"force,omitempty"`
	Predicate *string `json:"predicate,omitempty"`
	Extends   *string `json:"extends,omitempty"`
}
