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
		Mpn: "bt2000",
		Variants: []model.Variant{
			{VariantID: "a-corp/eagle/bt2000-variant.tm.jsonld"},
		},
	}

	inventoryEntry := mapper.GetInventoryEntry(entry)

	if assert.Len(t, inventoryEntry.HasVariant, 1) {
		assert.Equal(t, "a-corp/eagle/bt2000-variant.tm.jsonld", inventoryEntry.HasVariant[0].VariantId)
	}
}

func TestMapper_GetVariantsReturnsNilWhenEmpty(t *testing.T) {
	mapper := NewMapper(context.Background())

	assert.Nil(t, mapper.GetVariants(nil))
	assert.Nil(t, mapper.GetVariants([]model.Variant{}))
}
