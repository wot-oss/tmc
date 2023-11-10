package model

type ThingModel struct {
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Version      Version            `json:"version"`
}

type SchemaAuthor struct {
	Name string `json:"name" validate:"required"` // fixme: make url-friendly where used, esp. escape / and \
}
type SchemaManufacturer struct {
	Name string `json:"name" validate:"required"` // fixme: make url-friendly where used
}

type Version struct {
	Model string `json:"model"`
}
