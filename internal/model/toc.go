package model

import (
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
)

type TOC struct {
	Meta TOCMeta    `json:"meta"`
	Data []TOCEntry `json:"data"`
}

type TOCMeta struct {
	Created time.Time `json:"created"`
}

type TOCEntry struct {
	Name         string             `json:"name"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Versions     []TOCVersion       `json:"versions"`
}

type TOCVersion struct {
	ExtendedFields
	ID        string  `json:"tmid"`
	Digest    string  `json:"digest"`
	TimeStamp string  `json:"timestamp,omitempty"`
	Version   Version `json:"version"`
}

func (toc *TOC) Filter(filter string) {
	for index, value := range toc.Data {
		if !matchFilter(value, filter) {
			// zero the reference to make it garbage collected
			toc.Data[index] = TOCEntry{}
			toc.Data = append(toc.Data[:index], toc.Data[index+1:]...)
		}
	}
}

func matchFilter(entry TOCEntry, filter string) bool {
	filter = internal.Prep(filter)
	if strings.Contains(internal.Prep(entry.Name), filter) {
		return true
	}
	if strings.Contains(internal.Prep(entry.Manufacturer.Name), filter) {
		return true
	}
	if strings.Contains(internal.Prep(entry.Mpn), filter) {
		return true
	}
	for _, version := range entry.Versions {
		if strings.Contains(internal.Prep(version.Description), filter) {
			return true
		}
	}
	return false
}

func (toc *TOC) FindByName(name string) (ent TOCEntry, ok bool) {
	for _, value := range toc.Data {
		if value.Name == name {
			return value, true
		}
	}
	return TOCEntry{}, false
}
