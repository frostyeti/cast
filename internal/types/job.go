package types

type Job struct {
	Id    string   `json:"id"`
	Name  *string  `json:"name,omitempty"`
	Desc  *string  `json:"desc,omitempty"`
	Needs []string `json:"needs,omitempty"`
	Steps []Step   `json:"steps,omitempty"`
	If    *string  `json:"if,omitempty"`
}
