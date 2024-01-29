package model

import "github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"

type TOCToSearchResultMapper struct {
	foundIn FoundSource
}

func NewTOCToFoundMapper(s FoundSource) *TOCToSearchResultMapper {
	return &TOCToSearchResultMapper{foundIn: s}
}

func (m *TOCToSearchResultMapper) ToSearchResult(toc TOC) SearchResult {
	r := SearchResult{}
	var es []FoundEntry
	for _, e := range toc.Data {
		es = append(es, m.ToFoundEntry(e))
	}
	r.Entries = es
	return r
}

func (m *TOCToSearchResultMapper) ToFoundEntry(e *TOCEntry) FoundEntry {
	return FoundEntry{
		Name:         e.Name,
		Manufacturer: e.Manufacturer,
		Mpn:          e.Mpn,
		Author:       e.Author,
		Versions:     m.ToFoundVersions(e.Versions),
	}
}

func (m *TOCToSearchResultMapper) ToFoundVersions(versions []TOCVersion) []FoundVersion {
	var r []FoundVersion
	for _, v := range versions {
		r = append(r, FoundVersion{
			TOCVersion: v,
			FoundIn:    m.foundIn,
		})
	}
	return r
}

type InventoryResponseToSearchResultMapper struct {
	foundIn     FoundSource
	linksMapper func(links server.InventoryEntryVersion) map[string]string
}

func NewInventoryResponseToSearchResultMapper(s FoundSource, linksMapper func(links server.InventoryEntryVersion) map[string]string) *InventoryResponseToSearchResultMapper {
	return &InventoryResponseToSearchResultMapper{foundIn: s, linksMapper: linksMapper}
}

func (m *InventoryResponseToSearchResultMapper) ToSearchResult(inv server.InventoryResponse) SearchResult {
	r := SearchResult{}
	var es []FoundEntry
	for _, e := range inv.Data {
		es = append(es, m.ToFoundEntry(e))
	}
	r.Entries = es
	return r
}

func (m *InventoryResponseToSearchResultMapper) ToFoundEntry(e server.InventoryEntry) FoundEntry {
	return FoundEntry{
		Name:         e.Name,
		Manufacturer: SchemaManufacturer{Name: e.SchemaManufacturer.SchemaName},
		Mpn:          e.SchemaMpn,
		Author:       SchemaAuthor{Name: e.SchemaAuthor.SchemaName},
		Versions:     m.ToFoundVersions(e.Versions),
	}
}

func (m *InventoryResponseToSearchResultMapper) ToFoundVersions(versions []server.InventoryEntryVersion) []FoundVersion {
	var r []FoundVersion
	for _, v := range versions {
		r = append(r, FoundVersion{
			TOCVersion: TOCVersion{
				Description: v.Description,
				Version:     Version{Model: v.Version.Model},
				Links:       m.ToFoundVersionLinks(v),
				TMID:        v.TmID,
				Digest:      v.Digest,
				TimeStamp:   v.Timestamp,
				ExternalID:  v.ExternalID,
			},
			FoundIn: m.foundIn,
		})
	}
	return r
}

func (m *InventoryResponseToSearchResultMapper) ToFoundVersionLinks(v server.InventoryEntryVersion) map[string]string {
	if m.linksMapper != nil {
		return m.linksMapper(v)
	}
	if v.Links == nil {
		return nil
	}
	return map[string]string{
		"content": v.Links.Content,
	}
}
