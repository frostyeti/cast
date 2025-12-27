package models

type Need struct {
	Id       string
	Parallel bool
}

type Needs []Need

func (needs *Needs) Ids() []string {
	names := make([]string, len(*needs))
	for i, need := range *needs {
		names[i] = need.Id
	}
	return names
}
