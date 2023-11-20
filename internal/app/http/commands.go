package http

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"slices"
)

type FilterParams struct {
	Author       []string
	Manufacturer []string
	Mpn          []string
	Original     []string
}

type SearchParams struct {
	query string
}

func Filter(toc *model.Toc, filter *FilterParams) {
	if filter == nil || toc == nil {
		return
	}

	for name, tocEntry := range toc.Contents {
		if !filterMatches(filter.Author, tocEntry.Author.Name) {
			delete(toc.Contents, name)
			continue
		}
		if !filterMatches(filter.Manufacturer, tocEntry.Manufacturer.Name) {
			delete(toc.Contents, name)
			continue
		}
		if !filterMatches(filter.Mpn, tocEntry.Mpn) {
			delete(toc.Contents, name)
			continue
		}
		if filter.Original != nil && len(filter.Original) > 0 {
			hasOriginal := false
			for _, v := range tocEntry.Versions {
				if slices.Contains(filter.Original, v.Original) {
					hasOriginal = true
					break
				}
			}
			if !hasOriginal {
				delete(toc.Contents, name)
			}
		}
	}
}

func filterMatches(filterValues []string, value string) bool {
	if filterValues == nil || len(filterValues) == 0 {
		return true
	}
	return slices.Contains(filterValues, value)
}

// todo: should return error if e.g. content search library fails
func Search(toc *model.Toc, search *SearchParams) {
	if search == nil || toc == nil {
		return
	}
	toc.Filter(search.query)
}
