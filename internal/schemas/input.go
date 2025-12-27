package schemas

type Input struct {
	Id        string   `json:"id"`
	Desc      *string  `json:"description,omitempty"`
	Default   *string  `json:"default,omitempty"`
	Required  *bool    `json:"required,omitempty"`
	Type      *string  `json:"type,omitempty"`
	Selection []string `json:"selection,omitempty"`
}
