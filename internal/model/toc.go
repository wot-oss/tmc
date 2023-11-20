package model

import (
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
)

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

func (toc *Toc) Filter(filter string) {
	for name, value := range toc.Contents {
		if !matchFilter(name, value, filter) {
			delete(toc.Contents, name)
		}
	}
}

func matchFilter(name string, thing TocThing, filter string) bool {
	filter = internal.Prep(filter)
	if strings.Contains(internal.Prep(name), filter) {
		return true
	}
	if strings.Contains(internal.Prep(thing.Manufacturer.Name), filter) {
		return true
	}
	if strings.Contains(internal.Prep(thing.Mpn), filter) {
		return true
	}
	for _, version := range thing.Versions {
		if strings.Contains(internal.Prep(version.Description), filter) {
			return true
		}
	}
	return false
}
