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
	ThingModel
	Versions []TocVersion `json:"versions"`
}

type TocVersion struct {
	ExtendedFields
	TimeStamp *time.Time `json:"timestamp,omitempty"`
}
