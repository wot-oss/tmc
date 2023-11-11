package model

import "time"

type Toc struct {
	Meta     TocMeta             `json:"meta"`
	Contents map[string]TocThing `json:"contents"`
}

type TocMeta struct {
	Created time.Time `json:"created"`
}

type TocThing struct {
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Versions     []TocVersion       `json:"versions"`
}

type TocVersion struct {
	ExtendedFields
	TimeStamp string  `json:"timestamp,omitempty"`
	Version   Version `json:"version"`
}
