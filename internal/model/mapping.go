package model

import (
	"fmt"

	"github.com/wot-oss/tmc/internal/app/http/server"
)

type IndexToSearchResultMapper struct {
	foundIn FoundSource
}

func NewIndexToFoundMapper(s FoundSource) *IndexToSearchResultMapper {
	return &IndexToSearchResultMapper{foundIn: s}
}

func (m *IndexToSearchResultMapper) ToSearchResult(idx Index) SearchResult {
	r := SearchResult{}
	var es []FoundEntry
	for _, e := range idx.Data {
		es = append(es, m.ToFoundEntry(e))
	}
	r.Entries = es
	return r
}

func (m *IndexToSearchResultMapper) ToFoundEntry(e *IndexEntry) FoundEntry {
	return FoundEntry{
		Name:                e.Name,
		Manufacturer:        e.Manufacturer,
		Mpn:                 e.Mpn,
		Author:              e.Author,
		Versions:            m.ToFoundVersions(e.Versions),
		FoundIn:             m.foundIn,
		AttachmentContainer: e.AttachmentContainer,
	}
}

func (m *IndexToSearchResultMapper) ToFoundVersions(versions []IndexVersion) []FoundVersion {
	var r []FoundVersion
	for _, v := range versions {
		r = append(r, m.ToFoundVersion(v))
	}
	return r
}

func (m *IndexToSearchResultMapper) ToFoundVersion(v IndexVersion) FoundVersion {
	return FoundVersion{
		IndexVersion: v,
		FoundIn:      m.foundIn,
	}
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
		Name:         e.TmName,
		Manufacturer: SchemaManufacturer{Name: e.SchemaManufacturer.SchemaName},
		Mpn:          e.SchemaMpn,
		Author:       SchemaAuthor{Name: e.SchemaAuthor.SchemaName},
		Versions:     m.ToFoundVersions(e.Versions),
		AttachmentContainer: AttachmentContainer{
			Attachments: m.ToFoundVersionAttachments(e.Attachments),
		},
		FoundIn: m.subRepoFoundSource(e.Source),
	}
}

func (m *InventoryResponseToSearchResultMapper) ToFoundVersions(versions []server.InventoryEntryVersion) []FoundVersion {
	var r []FoundVersion
	for _, v := range versions {
		r = append(r, m.ToFoundVersion(v))
	}
	return r
}

func (m *InventoryResponseToSearchResultMapper) ToFoundVersion(v server.InventoryEntryVersion) FoundVersion {
	version := FoundVersion{
		IndexVersion: IndexVersion{
			Description: v.Description,
			Version:     Version{Model: v.Version.Model},
			Links:       m.ToFoundVersionLinks(v),
			TMID:        v.TmID,
			Digest:      v.Digest,
			TimeStamp:   v.Timestamp,
			ExternalID:  v.ExternalID,
			AttachmentContainer: AttachmentContainer{
				Attachments: m.ToFoundVersionAttachments(v.Attachments),
			},
		},
		FoundIn: m.subRepoFoundSource(v.Source),
	}
	return version
}

func (m *InventoryResponseToSearchResultMapper) ToFoundVersionAttachments(al *server.AttachmentsList) []Attachment {
	if al == nil {
		return nil
	}
	var atts []Attachment
	for _, a := range *al {
		att := Attachment{
			Name: a.Name,
		}
		atts = append(atts, att)
	}
	return atts
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

func (m *InventoryResponseToSearchResultMapper) subRepoFoundSource(source *server.SourceRepository) FoundSource {
	fi := m.foundIn
	if fi.RepoName != "" && source != nil && *source != "" {
		fi = FoundSource{RepoName: fmt.Sprintf("%s/%s", fi.RepoName, *source)}
	}
	return fi
}
