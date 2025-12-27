package models

type CyclicalReferenceError struct {
	Cycles []Task
}

func (e *CyclicalReferenceError) Error() string {
	msg := "Cyclical references found in tasks:\n"
	for _, cycle := range e.Cycles {
		msg += " - " + cycle.Id + "\n"
	}
	return msg
}
