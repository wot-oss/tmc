package http

import (
	"slices"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type FilterParams struct {
	Author       []string
	Manufacturer []string
	Mpn          []string
	ExternalID   []string
}

type SearchParams struct {
	query string
}

func Filter(toc *model.TOC, filter *FilterParams) {
	if filter == nil || toc == nil {
		return
	}

	toc.Data = slices.DeleteFunc(toc.Data, func(tocEntry *model.TOCEntry) bool {
		if !filterMatches(filter.Author, tocEntry.Author.Name) {
			return true
		}

		if !filterMatches(filter.Manufacturer, tocEntry.Manufacturer.Name) {
			return true
		}

		if !filterMatches(filter.Mpn, tocEntry.Mpn) {
			return true
		}

		if filter.ExternalID != nil && len(filter.ExternalID) > 0 {
			hasExternalID := false
			for _, v := range tocEntry.Versions {
				if slices.Contains(filter.ExternalID, v.ExternalID) {
					hasExternalID = true
					break
				}
			}
			if !hasExternalID {
				return true
			}
		}
		return false
	})
}

func filterMatches(filterValues []string, value string) bool {
	if filterValues == nil || len(filterValues) == 0 {
		return true
	}
	return slices.Contains(filterValues, value)
}

// todo: should return error if e.g. content search library fails
func Search(toc *model.TOC, search *SearchParams) {
	if search == nil || toc == nil {
		return
	}
	toc.Filter(search.query)
}
