package models

type AccessorObjects []AccessorObject

func (t AccessorObjects) AsMap() map[string]any {
	output := make(map[string]any)
	for _, accessor := range t {
		output[accessor.Name] = accessor.Data
	}

	return output
}

type AccessorObject struct {
	Name string
	Data any
}
