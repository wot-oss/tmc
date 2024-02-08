package http

import (
	"context"
	"net/url"
	"strings"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

type Mapper struct {
	Ctx context.Context
}

func NewMapper(ctx context.Context) *Mapper {
	return &Mapper{
		Ctx: ctx,
	}
}

func (m *Mapper) GetInventoryMeta(toc model.SearchResult) server.Meta {
	return server.Meta{
		Page: &server.MetaPage{
			Elements: len(toc.Entries),
		},
	}
}

func (m *Mapper) GetInventoryData(tocData []model.FoundEntry) []server.InventoryEntry {
	data := []server.InventoryEntry{}
	for _, v := range tocData {
		data = append(data, m.GetInventoryEntry(v))
	}

	return data
}

func (m *Mapper) GetInventoryEntry(tocEntry model.FoundEntry) server.InventoryEntry {
	invEntry := server.InventoryEntry{}
	invEntry.Name = tocEntry.Name
	invEntry.SchemaAuthor.SchemaName = tocEntry.Author.Name
	invEntry.SchemaManufacturer.SchemaName = tocEntry.Manufacturer.Name
	invEntry.SchemaMpn = tocEntry.Mpn
	invEntry.Versions = m.GetInventoryEntryVersions(tocEntry.Versions)

	hrefSelf, _ := url.JoinPath(basePathInventory, tocEntry.Name)
	hrefSelf = resolveRelativeLink(m.Ctx, hrefSelf)
	links := server.InventoryEntryLinks{
		Self: hrefSelf,
	}

	invEntry.Links = &links

	return invEntry
}

func (m *Mapper) GetInventoryEntryVersions(tocVersions []model.FoundVersion) []server.InventoryEntryVersion {
	invVersions := []server.InventoryEntryVersion{}
	for _, v := range tocVersions {
		invVersion := m.GetInventoryEntryVersion(v)
		invVersions = append(invVersions, invVersion)
	}

	return invVersions
}

func (m *Mapper) GetInventoryEntryVersion(tocVersion model.FoundVersion) server.InventoryEntryVersion {
	invVersion := server.InventoryEntryVersion{}

	invVersion.TmID = tocVersion.TMID
	invVersion.Version.Model = tocVersion.Version.Model
	invVersion.ExternalID = tocVersion.ExternalID
	invVersion.Description = tocVersion.Description
	invVersion.Timestamp = tocVersion.TimeStamp
	invVersion.Digest = tocVersion.Digest

	hrefContent, _ := url.JoinPath(basePathThingModels, tocVersion.TMID)
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
