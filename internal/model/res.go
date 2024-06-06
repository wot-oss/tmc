package model

const (
	ResTypeUnknown ResourceType = iota
	ResTypeTM
)

type ResourceType int

type ResourceFilter struct {
	Types []ResourceType
	Names []string
}

type Resource struct {
	Name    string
	RelPath string
	Typ     ResourceType
	Raw     []byte
}
