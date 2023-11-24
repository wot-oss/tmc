package http

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"net/url"
)

func mapInventory(toc model.Toc) Inventory {
	inv := Inventory{}
	inv.Meta.Created = toc.Meta.Created
	inv.Contents = mapInventoryContents(toc.Contents)

	return inv
}

func mapInventoryContents(tocContent map[string]model.TocThing) map[string]InventoryEntry {
	content := map[string]InventoryEntry{}
	for k, v := range tocContent {
		content[k] = mapInventoryEntry(k, v)
	}

	return content
}

func mapInventoryEntry(tocEntryId string, tocThing model.TocThing) InventoryEntry {
	invEntry := InventoryEntry{}
	invEntry.SchemaAuthor.SchemaName = tocThing.Author.Name
	invEntry.SchemaManufacturer.SchemaName = tocThing.Manufacturer.Name
	invEntry.SchemaMpn = tocThing.Mpn
	invEntry.Versions = mapInvtoryEntryVersions(tocThing.Versions)

	var links []Link
	hrefSelf, _ := url.JoinPath("/inventory", tocEntryId)
	linkSelf := Link{
		Rel:  Self,
		Href: hrefSelf,
	}
	links = append(links, linkSelf)
	invEntry.Links = &links

	return invEntry
}

func mapInvtoryEntryVersions(tocVersions []model.TocVersion) []InventoryEntryVersion {
	invVersions := []InventoryEntryVersion{}
	for _, v := range tocVersions {
		invVersion := mapInventoryEntryVersion(v)
		invVersions = append(invVersions, invVersion)
	}

	return invVersions
}

func mapInventoryEntryVersion(tocVersion model.TocVersion) InventoryEntryVersion {
	invVersion := InventoryEntryVersion{}

	invVersion.TmId = tocVersion.ID
	invVersion.Version.Model = tocVersion.Version.Model
	invVersion.Original = tocVersion.Original
	invVersion.Description = tocVersion.Description
	invVersion.Timestamp = &tocVersion.TimeStamp

	var links []Link
	hrefContent, _ := url.JoinPath("/thing-models", tocVersion.ID)
	linkContent := Link{
		Rel:  Content,
		Href: hrefContent,
	}
	links = append(links, linkContent)

	invVersion.Links = &links

	return invVersion
}
