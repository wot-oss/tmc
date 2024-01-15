package model

import "github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"

// ThingModel is a model for unmarshalling a Thing Model to be
// imported. It contains only the fields required to be accepted into
// the catalog.
type ThingModel struct {
	ID           string             `json:"id,omitempty"`
	Description  string             `json:"description"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Version      Version            `json:"version"`
	Links        `json:"links"`
}

func (tm *ThingModel) IsOfficial() bool {
	return EqualsAsSchemaName(tm.Manufacturer.Name, tm.Author.Name)
}

func EqualsAsSchemaName(s1, s2 string) bool {
	return utils.ToTrimmedLower(s1) == utils.ToTrimmedLower(s2)
}

type SchemaAuthor struct {
	Name string `json:"schema:name" validate:"required"`
}
type SchemaManufacturer struct {
	Name string `json:"schema:name" validate:"required"`
}

type Version struct {
	Model string `json:"model"`
}
