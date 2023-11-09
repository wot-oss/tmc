package model

// ThingModel is a model for unmarshalling a Thing Model to be
// imported. It contains only the fields required to be accepted into
// the catalog.
type ThingModel struct {
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
}

type SchemaAuthor struct {
	Name string `json:"name" validate:"required"`
}
type SchemaManufacturer struct {
	Name string `json:"name" validate:"required"`
}

// CatalogThingModel is a model designed for the unmarshalling of a
// cataloged Thing Model. A cataloged Thing Model includes supplementary
// fields beyond the essential ones required for import, which have been
// introduced during the importing process.
type ExtendedFields struct {
	Path        string      `json:"path"`
	ID          string      `json:"id,omitempty"`
	Original    string      `json:"original"`
	Version     VersionInfo `json:"version"`
	Description string      `json:"description"`
}

type VersionInfo struct {
	Model string `json:"model"`
}

type CatalogThingModel struct {
	ThingModel
	ExtendedFields
}
