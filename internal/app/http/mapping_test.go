package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
)

func TestMapper_GetInventoryEntryIncludesVariants(t *testing.T) {
	mapper := NewMapper(context.Background())

	entry := model.FoundEntry{
		Name: "a-corp/eagle/bt2000",
		Author: model.SchemaAuthor{
			Name: "a-corp",
		},
		Manufacturer: model.SchemaManufacturer{
			Name: "eagle",
		},
		Mpn:         "bt2000",
		IsVariantOf: "a-corp/eagle/bt1000/v1.0.0-20240108140117-243d1b462ccd.tm.json",
		Variants: []model.Variant{
			{VariantID: "a-corp/eagle/bt2000-variant.tm.jsonld"},
		},
	}

	inventoryEntry := mapper.GetInventoryEntry(entry)

	if assert.Len(t, inventoryEntry.HasVariant, 1) {
		assert.Equal(t, "a-corp/eagle/bt2000-variant.tm.jsonld", inventoryEntry.HasVariant[0].VariantId)
	}
	assert.Equal(t, "a-corp/eagle/bt1000/v1.0.0-20240108140117-243d1b462ccd.tm.json", inventoryEntry.IsVariantOf)
}

func TestMapper_GetInventoryDataRemovesVariantChildrenFromParentHasVariant(t *testing.T) {
	mapper := NewMapper(context.Background())

	entries := []model.FoundEntry{
		{
			Name: "a-corp/eagle/bt2000",
			Author: model.SchemaAuthor{
				Name: "a-corp",
			},
			Manufacturer: model.SchemaManufacturer{
				Name: "eagle",
			},
			Mpn: "bt2000",
			Variants: []model.Variant{
				{VariantID: "a-corp/eagle/bt2000-variant/v1.0.0-20240108140117-243d1b462ccd.tm.json"},
				{VariantID: "a-corp/eagle/bt2000-standalone/v1.0.0-20240108140117-243d1b462ccd.tm.json"},
			},
		},
		{
			Name:        "a-corp/eagle/bt2000-variant",
			IsVariantOf: "a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccd.tm.json",
			Versions: []model.FoundVersion{{IndexVersion: &model.IndexVersion{TMID: "a-corp/eagle/bt2000-variant/v1.0.0-20240108140117-243d1b462ccd.tm.json"}}},
		},
	}

	inventoryEntries := mapper.GetInventoryData(entries)

	if assert.Len(t, inventoryEntries, 2) {
		assert.Equal(t, "a-corp/eagle/bt2000", inventoryEntries[0].TmName)
		if assert.Len(t, inventoryEntries[0].HasVariant, 1) {
			assert.Equal(t, "a-corp/eagle/bt2000-standalone/v1.0.0-20240108140117-243d1b462ccd.tm.json", inventoryEntries[0].HasVariant[0].VariantId)
		}
		assert.Equal(t, "a-corp/eagle/bt2000-variant", inventoryEntries[1].TmName)
		assert.Equal(t, "a-corp/eagle/bt2000/v1.0.0-20240108140117-243d1b462ccd.tm.json", inventoryEntries[1].IsVariantOf)
	}
}

func TestMapper_GetVariantsReturnsNilWhenEmpty(t *testing.T) {
	mapper := NewMapper(context.Background())

	assert.Nil(t, mapper.GetVariants(nil))
	assert.Nil(t, mapper.GetVariants([]model.Variant{}))
}
