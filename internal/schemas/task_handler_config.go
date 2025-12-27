package schemas

type TaskHandlerConfig struct {
	Id   string  `json:"id"`
	Name string  `json:"name"`
	Desc *string `json:"description,omitempty"`

	// with is used for bun/deno/docker
	With    *With    `json:"with,omitempty"`
	Inputs  []Input  `json:"inputs,omitempty"`
	Outputs []Output `json:"outputs,omitempty"`
	Uses    *string  `json:"uses,omitempty"`
	Run     *string  `json:"run,omitempty"`
	Steps   []Step   `json:"steps,omitempty"`
}
