package model

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
