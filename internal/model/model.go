package model

// ThingModel is a model for unmarshalling a Thing Model to be
// imported. It contains only the fields required to be accepted into
// the catalog.
type ThingModel struct {
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Version      Version            `json:"version"`
}

type SchemaAuthor struct {
	Name string `json:"name" validate:"required"`
}
type SchemaManufacturer struct {
	Name string `json:"name" validate:"required"`
}

type Version struct {
	Model string `json:"model"`
}

type ExtendedFields struct {
	ID          string `json:"id,omitempty"`
	Original    string `json:"original"`
	Description string `json:"description"`
	Path        string `json:"path"`
}

type CatalogThingModel struct {
	ThingModel
	ExtendedFields
}
