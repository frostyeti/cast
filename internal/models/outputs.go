package models

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/frostyeti/cast/internal/errors"
)

type Outputs map[string]any

func NewOutputs() Outputs {
	return make(Outputs)
}

func (o Outputs) ToMap() map[string]any {
	result := make(map[string]any)
	for k, v := range o {
		result[k] = v
	}
	return result
}

func (o Outputs) Clone() Outputs {
	clone := make(Outputs)
	for k, v := range o {
		clone[k] = v
	}
	return clone
}

func (o Outputs) Get(key string) any {
	if !strings.ContainsRune(key, '.') {
		if val, ok := o[key]; ok {
			return val
		}
		return nil
	}

	segments := strings.Split(key, ".")
	var current any = o
	for _, segment := range segments {
		if currentSlice, ok := current.([]any); ok {
			if index, err := strconv.Atoi(segment); err == nil {
				if index < 0 || index >= len(currentSlice) {
					return nil
				}
				current = currentSlice[index]
				continue
			}

			return nil
		}

		if currentMap, ok := current.(map[string]any); ok {
			if val, ok := currentMap[segment]; ok {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func (o Outputs) Set(key string, value any) error {
	if !strings.ContainsRune(key, '.') {
		o[key] = value
		return nil
	}

	segments := strings.Split(key, ".")
	var current any = o
	for i, segment := range segments {
		isLast := i == len(segments)-1

		if currentSlice, ok := current.([]any); ok {
			index, err := strconv.Atoi(segment)

			if err != nil {
				return fmt.Errorf("expected integer index for slice at segment '%s'", segment)
			}

			if index < 0 || index >= len(currentSlice) {
				return errors.Newf("index %d out of range for slice at segment '%s'", index, segment)
			}

			if isLast {
				currentSlice[index] = value
				return nil
			}

			current = currentSlice[index]
			continue
		}

		if currentMap, ok := current.(map[string]any); ok {
			if isLast {
				currentMap[segment] = value
				return nil
			}

			if next, ok := currentMap[segment]; ok {
				current = next
				continue
			} else {
				newMap := make(map[string]any)
				currentMap[segment] = newMap
				current = newMap
				continue
			}
		}

		return fmt.Errorf("cannot set value at segment '%s'", segment)
	}

	return nil
}

func (o *Outputs) Merge(other Outputs) {
	if o == nil {
		o = &Outputs{}
	}

	for k, v := range other {
		(*o)[k] = v
	}
}
