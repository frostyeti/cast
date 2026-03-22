package types

// Output describes a remote task output parameter.
type Output struct {
	Id      string  `json:"id"`
	Desc    *string `json:"description,omitempty"`
	Type    *string `json:"type,omitempty"`
	Default *string `json:"default,omitempty"`
}
