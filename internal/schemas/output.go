package schemas

type Output struct {
	Id      string  `json:"id"`
	Desc    *string `json:"description,omitempty"`
	Type    *string `json:"type,omitempty"`
	Default *string `json:"default,omitempty"`
}
