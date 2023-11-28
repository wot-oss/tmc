package http

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"net/url"
)

func mapInventoryMeta(toc model.TOC) Meta {
	meta := Meta{
		Created: toc.Meta.Created,
	}
	return meta
}

func mapInventoryData(tocData []*model.TOCEntry) []InventoryEntry {
	data := []InventoryEntry{}
	for _, v := range tocData {
		data = append(data, mapInventoryEntry(*v))
	}

	return data
}

func mapInventoryEntry(tocEntry model.TOCEntry) InventoryEntry {
	invEntry := InventoryEntry{}
	invEntry.Name = tocEntry.Name
	invEntry.SchemaAuthor.SchemaName = tocEntry.Author.Name
	invEntry.SchemaManufacturer.SchemaName = tocEntry.Manufacturer.Name
	invEntry.SchemaMpn = tocEntry.Mpn
	invEntry.Versions = mapInventoryEntryVersions(tocEntry.Versions)

	hrefSelf, _ := url.JoinPath(basePathInventory, tocEntry.Name)
	links := InventoryEntryLinks{
		Self: hrefSelf,
	}

	invEntry.Links = &links

	return invEntry
}

func mapInventoryEntryVersions(tocVersions []model.TOCVersion) []InventoryEntryVersion {
	invVersions := []InventoryEntryVersion{}
	for _, v := range tocVersions {
		invVersion := mapInventoryEntryVersion(v)
		invVersions = append(invVersions, invVersion)
	}

	return invVersions
}

func mapInventoryEntryVersion(tocVersion model.TOCVersion) InventoryEntryVersion {
	invVersion := InventoryEntryVersion{}

	invVersion.TmID = tocVersion.TMID
	invVersion.Version.Model = tocVersion.Version.Model
	invVersion.ExternalID = tocVersion.ExternalID
	invVersion.Description = tocVersion.Description
	invVersion.Timestamp = tocVersion.TimeStamp
	invVersion.Digest = tocVersion.Digest

	hrefContent, _ := url.JoinPath(basePathThingModels, tocVersion.TMID)
	links := InventoryEntryVersionLinks{
		Content: hrefContent,
	}

	invVersion.Links = &links

	return invVersion
}
