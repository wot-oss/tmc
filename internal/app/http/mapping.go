package http

import (
	"context"
	"net/url"
	"strings"

	"github.com/wot-oss/tmc/internal/app/http/server"
	"github.com/wot-oss/tmc/internal/model"
)

type Mapper struct {
	Ctx context.Context
}

func NewMapper(ctx context.Context) *Mapper {
	return &Mapper{
		Ctx: ctx,
	}
}

func (m *Mapper) GetInventoryMeta(res model.SearchResult) server.Meta {
	return server.Meta{
		Page: &server.MetaPage{
			Elements: len(res.Entries),
		},
	}
}

func (m *Mapper) GetInventoryData(entries []model.FoundEntry) []server.InventoryEntry {
	data := []server.InventoryEntry{}
	for _, v := range entries {
		data = append(data, m.GetInventoryEntry(v))
	}

	return data
}

func (m *Mapper) GetInventoryEntry(entry model.FoundEntry) server.InventoryEntry {
	invEntry := server.InventoryEntry{}
	invEntry.Name = entry.Name
	invEntry.SchemaAuthor.SchemaName = entry.Author.Name
	invEntry.SchemaManufacturer.SchemaName = entry.Manufacturer.Name
	invEntry.SchemaMpn = entry.Mpn
	invEntry.Versions = m.GetInventoryEntryVersions(entry.Versions)

	hrefSelf, _ := url.JoinPath(basePathInventory, entry.Name)
	hrefSelf = resolveRelativeLink(m.Ctx, hrefSelf)
	links := server.InventoryEntryLinks{
		Self: hrefSelf,
	}

	invEntry.Links = &links

	return invEntry
}

func (m *Mapper) GetInventoryEntryVersions(versions []model.FoundVersion) []server.InventoryEntryVersion {
	invVersions := []server.InventoryEntryVersion{}
	for _, v := range versions {
		invVersion := m.GetInventoryEntryVersion(v)
		invVersions = append(invVersions, invVersion)
	}

	return invVersions
}

func (m *Mapper) GetInventoryEntryVersion(version model.FoundVersion) server.InventoryEntryVersion {
	invVersion := server.InventoryEntryVersion{}

	invVersion.TmID = version.TMID
	invVersion.Version.Model = version.Version.Model
	invVersion.ExternalID = version.ExternalID
	invVersion.Description = version.Description
	invVersion.Timestamp = version.TimeStamp
	invVersion.Digest = version.Digest

	hrefContent, _ := url.JoinPath(basePathThingModels, version.TMID)
	hrefContent = resolveRelativeLink(m.Ctx, hrefContent)

	links := server.InventoryEntryVersionLinks{
		Content: hrefContent,
	}

	invVersion.Links = &links

	return invVersion
}

func resolveRelativeLink(ctx context.Context, link string) string {
	link, _ = strings.CutPrefix(link, "/")
	basePath := ctx.Value(ctxUrlRoot).(string)

	if basePath != "" {
		link, _ = url.JoinPath("/", basePath, link)
	} else {
		relDepth := ctx.Value(ctxRelPathDepth).(int)
		if relDepth <= 0 {
			link = "./" + link
		} else {
			link = strings.Repeat("../", relDepth) + link
		}
	}
	return link
}
