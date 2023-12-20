package http

import (
	"context"
	"net/url"
	"strings"

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

func (m *Mapper) GetInventoryMeta(toc model.SearchResult) Meta {
	meta := Meta{}
	return meta
}

func (m *Mapper) GetInventoryData(tocData []model.FoundEntry) []InventoryEntry {
	data := []InventoryEntry{}
	for _, v := range tocData {
		data = append(data, m.GetInventoryEntry(v))
	}

	return data
}

func (m *Mapper) GetInventoryEntry(tocEntry model.FoundEntry) InventoryEntry {
	invEntry := InventoryEntry{}
	invEntry.Name = tocEntry.Name
	invEntry.SchemaAuthor.SchemaName = tocEntry.Author.Name
	invEntry.SchemaManufacturer.SchemaName = tocEntry.Manufacturer.Name
	invEntry.SchemaMpn = tocEntry.Mpn
	invEntry.Versions = m.GetInventoryEntryVersions(tocEntry.Versions)

	hrefSelf, _ := url.JoinPath(basePathInventory, tocEntry.Name)
	hrefSelf = resolveRelativeLink(m.Ctx, hrefSelf)
	links := InventoryEntryLinks{
		Self: hrefSelf,
	}

	invEntry.Links = &links

	return invEntry
}

func (m *Mapper) GetInventoryEntryVersions(tocVersions []model.FoundVersion) []InventoryEntryVersion {
	invVersions := []InventoryEntryVersion{}
	for _, v := range tocVersions {
		invVersion := m.GetInventoryEntryVersion(v)
		invVersions = append(invVersions, invVersion)
	}

	return invVersions
}

func (m *Mapper) GetInventoryEntryVersion(tocVersion model.FoundVersion) InventoryEntryVersion {
	invVersion := InventoryEntryVersion{}

	invVersion.TmID = tocVersion.TMID
	invVersion.Version.Model = tocVersion.Version.Model
	invVersion.ExternalID = tocVersion.ExternalID
	invVersion.Description = tocVersion.Description
	invVersion.Timestamp = tocVersion.TimeStamp
	invVersion.Digest = tocVersion.Digest

	hrefContent, _ := url.JoinPath(basePathThingModels, tocVersion.TMID)
	hrefContent = resolveRelativeLink(m.Ctx, hrefContent)

	links := InventoryEntryVersionLinks{
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
