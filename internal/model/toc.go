package model

import (
	"strings"
	"time"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
)

type TOC struct {
	Meta TOCMeta     `json:"meta"`
	Data []*TOCEntry `json:"data"`
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

const TMLinkRel = "content"

type TOCVersion struct {
	Description string            `json:"description"`
	Version     Version           `json:"version"`
	Links       map[string]string `json:"links"`
	TMID        string            `json:"tmID"`
	Digest      string            `json:"digest"`
	TimeStamp   string            `json:"timestamp,omitempty"`
	ExternalID  string            `json:"externalID"`
}

func (toc *TOC) Filter(filter string) {
	for index, value := range toc.Data {
		if !matchFilter(*value, filter) {
			// zero the reference to make it garbage collected
			toc.Data[index] = &TOCEntry{}
			toc.Data = append(toc.Data[:index], toc.Data[index+1:]...)
		}
	}
}

func matchFilter(entry TOCEntry, filter string) bool {
	filter = internal.ToTrimmedLower(filter)
	if strings.Contains(internal.ToTrimmedLower(entry.Name), filter) {
		return true
	}
	if strings.Contains(internal.ToTrimmedLower(entry.Manufacturer.Name), filter) {
		return true
	}
	if strings.Contains(internal.ToTrimmedLower(entry.Mpn), filter) {
		return true
	}
	for _, version := range entry.Versions {
		if strings.Contains(internal.ToTrimmedLower(version.Description), filter) {
			return true
		}
	}
	return false
}

// FindByName searches by name and returns a reference to the TOCEnry if found
func (toc *TOC) FindByName(name string) *TOCEntry {
	for _, value := range toc.Data {
		if value.Name == name {
			return value
		}
	}
	return nil
}
