// Package http provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.0.0 DO NOT EDIT.
package http

import (
	"time"
)

// Defines values for GetInventoryParamsSort.
const (
	Author       GetInventoryParamsSort = "author"
	Manufacturer GetInventoryParamsSort = "manufacturer"
	Mpn          GetInventoryParamsSort = "mpn"
)

// AuthorsResponse defines model for AuthorsResponse.
type AuthorsResponse struct {
	Data []string `json:"data"`
}

// ErrorResponse defines model for ErrorResponse.
type ErrorResponse struct {
	Detail   *string `json:"detail,omitempty"`
	Instance *string `json:"instance,omitempty"`
	Status   int     `json:"status"`
	Title    string  `json:"title"`
	Type     *string `json:"type,omitempty"`
}

// InventoryEntry defines model for InventoryEntry.
type InventoryEntry struct {
	Name               string                  `json:"name"`
	SchemaAuthor       SchemaAuthor            `json:"schema:author"`
	SchemaManufacturer SchemaManufacturer      `json:"schema:manufacturer"`
	SchemaMpn          string                  `json:"schema:mpn"`
	Versions           []InventoryEntryVersion `json:"versions"`
	Links              *InventoryEntryLinks    `json:"links,omitempty"`
}

// InventoryEntryLinks defines model for InventoryEntryLinks.
type InventoryEntryLinks struct {
	Self string `json:"self"`
}

// InventoryEntryResponse defines model for InventoryEntryResponse.
type InventoryEntryResponse struct {
	Data InventoryEntry `json:"data"`
}

// InventoryEntryVersion defines model for InventoryEntryVersion.
type InventoryEntryVersion struct {
	Description string                      `json:"description"`
	Version     ModelVersion                `json:"version"`
	Links       *InventoryEntryVersionLinks `json:"links,omitempty"`
	TmID        string                      `json:"tmID"`
	Digest      string                      `json:"digest"`
	Timestamp   string                      `json:"timestamp"`
	ExternalID  string                      `json:"externalID"`
}

// InventoryEntryVersionLinks defines model for InventoryEntryVersionLinks.
type InventoryEntryVersionLinks struct {
	Content string `json:"content"`
}

// InventoryEntryVersionsResponse defines model for InventoryEntryVersionsResponse.
type InventoryEntryVersionsResponse struct {
	Data []InventoryEntryVersion `json:"data"`
}

// InventoryResponse defines model for InventoryResponse.
type InventoryResponse struct {
	Data []InventoryEntry `json:"data"`
	Meta *Meta            `json:"meta,omitempty"`
}

// ManufacturersResponse defines model for ManufacturersResponse.
type ManufacturersResponse struct {
	Data []string `json:"data"`
}

// Meta defines model for Meta.
type Meta struct {
	Created time.Time `json:"created"`
}

// ModelVersion defines model for ModelVersion.
type ModelVersion struct {
	Model string `json:"model"`
}

// MpnsResponse defines model for MpnsResponse.
type MpnsResponse struct {
	Data []string `json:"data"`
}

// PushThingModelResponse defines model for PushThingModelResponse.
type PushThingModelResponse struct {
	Data PushThingModelResult `json:"data"`
}

// PushThingModelResult defines model for PushThingModelResult.
type PushThingModelResult struct {
	TmID string `json:"tmID"`
}

// SchemaAuthor defines model for SchemaAuthor.
type SchemaAuthor struct {
	SchemaName string `json:"schema:name"`
}

// SchemaManufacturer defines model for SchemaManufacturer.
type SchemaManufacturer struct {
	SchemaName string `json:"schema:name"`
}

// GetAuthorsParams defines parameters for GetAuthors.
type GetAuthorsParams struct {
	// FilterManufacturer Filters the authors according to whether they have inventory entries
	// which belong to at least one of the given manufacturers with an exact match.
	// The filter works additive to other filters.
	FilterManufacturer *string `form:"filter.manufacturer,omitempty" json:"filter.manufacturer,omitempty"`

	// FilterMpn Filters the authors according to whether they have inventory entries
	// which belong to at least one of the given mpn (manufacturer part number) with an exact match.
	// The filter works additive to other filters.
	FilterMpn *string `form:"filter.mpn,omitempty" json:"filter.mpn,omitempty"`

	// FilterExternalID Filters the authors according to whether they have inventory entries
	// which belong to at least one of the given external ID's with an exact match.
	// The filter works additive to other filters.
	FilterExternalID *string `form:"filter.externalID,omitempty" json:"filter.externalID,omitempty"`

	// Search Filters the authors according to whether they have inventory entries
	// where their content matches the given search.
	// The search works additive to other filters.
	Search *string `form:"search,omitempty" json:"search,omitempty"`
}

// GetInventoryParams defines parameters for GetInventory.
type GetInventoryParams struct {
	// FilterAuthor Filters the inventory by one or more authors having exact match.
	// The filter works additive to other filters.
	FilterAuthor *string `form:"filter.author,omitempty" json:"filter.author,omitempty"`

	// FilterManufacturer Filters the inventory by one or more manufacturers having exact match.
	// The filter works additive to other filters.
	FilterManufacturer *string `form:"filter.manufacturer,omitempty" json:"filter.manufacturer,omitempty"`

	// FilterMpn Filters the inventory by one ore more mpn (manufacturer part number) having exact match.
	// The filter works additive to other filters.
	FilterMpn *string `form:"filter.mpn,omitempty" json:"filter.mpn,omitempty"`

	// FilterExternalID Filters the inventory by one or more external ID having exact match.
	// The filter works additive to other filters.
	FilterExternalID *string `form:"filter.externalID,omitempty" json:"filter.externalID,omitempty"`

	// Search Filters the inventory according to whether the content of the inventory entries matches the given search.
	// The search works additive to other filters.
	Search *string `form:"search,omitempty" json:"search,omitempty"`

	// Sort Sorts the inventory by one or more fields. The sort is applied in the order of the fields.
	// The sorting is done ascending per field by default. If a field needs to be sorted descending,
	// prefix it with a HYPHEN-MINUS "-")
	Sort *GetInventoryParamsSort `form:"sort,omitempty" json:"sort,omitempty"`
}

// GetInventoryParamsSort defines parameters for GetInventory.
type GetInventoryParamsSort string

// GetManufacturersParams defines parameters for GetManufacturers.
type GetManufacturersParams struct {
	// FilterAuthor Filters the manufacturers according to whether they belong to at least one of the given authors with an exact match.
	// The filter works additive to other filters.
	FilterAuthor *string `form:"filter.author,omitempty" json:"filter.author,omitempty"`

	// FilterMpn Filters the manufacturers according to whether they have inventory entries
	// which belong to at least one of the given mpn (manufacturer part number) with an exact match.
	// The filter works additive to other filters.
	FilterMpn *string `form:"filter.mpn,omitempty" json:"filter.mpn,omitempty"`

	// FilterExternalID Filters the manufacturers according to whether they have inventory entries
	// which belong to at least one of the given external ID's with an exact match.
	// The filter works additive to other filters.
	FilterExternalID *string `form:"filter.externalID,omitempty" json:"filter.externalID,omitempty"`

	// Search Filters the manufacturers according to whether they have inventory entries
	// where their content matches the given search.
	// The search works additive to other filters.
	Search *string `form:"search,omitempty" json:"search,omitempty"`
}

// GetMpnsParams defines parameters for GetMpns.
type GetMpnsParams struct {
	// FilterAuthor Filters the mpns according to whether they belong to at least one of the given authors with an exact match.
	// The filter works additive to other filters.
	FilterAuthor *string `form:"filter.author,omitempty" json:"filter.author,omitempty"`

	// FilterManufacturer Filters the mpns according to whether they belong to at least one of the given manufacturers with an exact match.
	// The filter works additive to other filters.
	FilterManufacturer *string `form:"filter.manufacturer,omitempty" json:"filter.manufacturer,omitempty"`

	// FilterExternalID Filters the mpns according to whether their inventory entry
	// belongs to at least one of the given external ID's with an exact match.
	// The filter works additive to other filters.
	FilterExternalID *string `form:"filter.externalID,omitempty" json:"filter.externalID,omitempty"`

	// Search Filters the mpns according to whether their inventory entry content matches the given search.
	// The search works additive to other filters.
	Search *string `form:"search,omitempty" json:"search,omitempty"`
}

// PushThingModelJSONBody defines parameters for PushThingModel.
type PushThingModelJSONBody = map[string]interface{}

// PushThingModelJSONRequestBody defines body for PushThingModel for application/json ContentType.
type PushThingModelJSONRequestBody = PushThingModelJSONBody
