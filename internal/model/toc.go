package model

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/wot-oss/tmc/internal/utils"
)

type Index struct {
	Meta IndexMeta     `json:"meta"`
	Data []*IndexEntry `json:"data"`
}

type IndexMeta struct {
	Created time.Time `json:"created"`
}

type IndexEntry struct {
	Name         string             `json:"name"`
	Manufacturer SchemaManufacturer `json:"schema:manufacturer" validate:"required"`
	Mpn          string             `json:"schema:mpn" validate:"required"`
	Author       SchemaAuthor       `json:"schema:author" validate:"required"`
	Versions     []IndexVersion     `json:"versions"`
}

func (e *IndexEntry) MatchesSearchText(searchQuery string) bool {
	if e == nil {
		return false
	}
	searchQuery = utils.ToTrimmedLower(searchQuery)
	if strings.Contains(utils.ToTrimmedLower(e.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Author.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Manufacturer.Name), searchQuery) {
		return true
	}
	if strings.Contains(utils.ToTrimmedLower(e.Mpn), searchQuery) {
		return true
	}
	for _, version := range e.Versions {
		if strings.Contains(utils.ToTrimmedLower(version.Description), searchQuery) {
			return true
		}
		if strings.Contains(utils.ToTrimmedLower(version.ExternalID), searchQuery) {
			return true
		}
	}
	return false

}

const TMLinkRel = "content"

type IndexVersion struct {
	Description string            `json:"description"`
	Version     Version           `json:"version"`
	Links       map[string]string `json:"links"`
	TMID        string            `json:"tmID"`
	Digest      string            `json:"digest"`
	TimeStamp   string            `json:"timestamp,omitempty"`
	ExternalID  string            `json:"externalID"`
}

func (idx *Index) Filter(search *SearchParams) {
	if search == nil {
		return
	}
	search.Sanitize()
	idx.Data = slices.DeleteFunc(idx.Data, func(entry *IndexEntry) bool {
		if !entry.MatchesSearchText(search.Query) {
			return true
		}

		if !matchesNameFilter(search.Name, entry.Name, search.Options) {
			return true
		}

		if !matchesFilter(search.Author, entry.Author.Name) {
			return true
		}

		if !matchesFilter(search.Manufacturer, entry.Manufacturer.Name) {
			return true
		}

		if !matchesFilter(search.Mpn, entry.Mpn) {
			return true
		}

		return false
	})

}

func matchesNameFilter(acceptedValue string, value string, options SearchOptions) bool {
	if len(acceptedValue) == 0 {
		return true
	}

	switch options.NameFilterType {
	case FullMatch:
		return value == acceptedValue
	case PrefixMatch:
		actualPathParts := strings.Split(value, "/")
		acceptedValue = strings.Trim(acceptedValue, "/")
		acceptedPathParts := strings.Split(acceptedValue, "/")
		if len(acceptedPathParts) > len(actualPathParts) {
			return false
		}
		return slices.Equal(actualPathParts[0:len(acceptedPathParts)], acceptedPathParts)
	default:
		panic(fmt.Sprintf("unsupported NameFilterType: %d", options.NameFilterType))
	}
}

func matchesFilter(acceptedValues []string, value string) bool {
	if len(acceptedValues) == 0 {
		return true
	}
	return slices.Contains(acceptedValues, utils.SanitizeName(value))
}

// findByName searches by name and returns a pointer to the IndexEntry if found
func (idx *Index) findByName(name string) *IndexEntry {
	for _, value := range idx.Data {
		if value.Name == name {
			return value
		}
	}
	return nil
}

// Insert uses CatalogThingModel to add a version, either to an existing
// entry or as a new entry. Returns the TMID of the inserted entry
func (idx *Index) Insert(ctm *ThingModel) (TMID, error) {
	tmid, err := ParseTMID(ctm.ID)
	if err != nil {
		return TMID{}, err
	}
	// find the right entry, or create if it doesn't exist
	idxEntry := idx.findByName(tmid.Name)
	if idxEntry == nil {
		idxEntry = &IndexEntry{
			Name:         tmid.Name,
			Manufacturer: SchemaManufacturer{Name: ctm.Manufacturer.Name},
			Mpn:          ctm.Mpn,
			Author:       SchemaAuthor{Name: ctm.Author.Name},
		}
		idx.Data = append(idx.Data, idxEntry)
	}
	// TODO: check if id already exists?
	// Append version information to entry
	externalID := ""
	original := ctm.Links.FindLink("original")
	if original != nil {
		externalID = original.HRef
	}
	tv := IndexVersion{
		Description: ctm.Description,
		TimeStamp:   tmid.Version.Timestamp,
		Version:     Version{Model: tmid.Version.Base.String()},
		TMID:        ctm.ID,
		ExternalID:  externalID,
		Digest:      tmid.Version.Hash,
		Links:       map[string]string{"content": tmid.String()},
	}
	if idx := slices.IndexFunc(idxEntry.Versions, func(version IndexVersion) bool {
		return version.TMID == ctm.ID
	}); idx == -1 {
		idxEntry.Versions = append(idxEntry.Versions, tv)
	} else {
		idxEntry.Versions[idx] = tv
	}
	return tmid, nil
}

// Delete deletes the record for the given id. Returns TM name to be removed from names file if no more versions are left
func (idx *Index) Delete(id string) (updated bool, deletedName string, err error) {
	var idxEntry *IndexEntry

	name, found := strings.CutSuffix(id, "/"+filepath.Base(id))
	if !found {
		return false, "", ErrInvalidId
	}
	idxEntry = idx.findByName(name)
	if idxEntry != nil {
		idxEntry.Versions = slices.DeleteFunc(idxEntry.Versions, func(version IndexVersion) bool {
			fnd := version.TMID == id
			if fnd {
				updated = true
			}
			return fnd
		})
		if len(idxEntry.Versions) == 0 {
			idx.Data = slices.DeleteFunc(idx.Data, func(entry *IndexEntry) bool {
				return entry.Name == name
			})
			return updated, name, nil
		}
	}
	return updated, "", nil
}
