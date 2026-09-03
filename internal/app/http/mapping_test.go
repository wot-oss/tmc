package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wot-oss/tmc/internal/model"
)

func TestMapper_GetInventoryEntryIncludesFamily(t *testing.T) {
	mapper := NewMapper(context.Background())

	entry := model.FoundEntry{
		Name: "a-corp/eagle/bt2000",
		Author: model.SchemaAuthor{
			Name: "a-corp",
		},
		Manufacturer: model.SchemaManufacturer{
			Name: "eagle",
		},
		Mpn:      "bt2000",
		FamilyID: "family-1",
	}

	inventoryEntry := mapper.GetInventoryEntry(entry)

	if assert.NotNil(t, inventoryEntry.Family) {
		assert.Equal(t, "family-1", *inventoryEntry.Family)
	}
}

func TestMapper_GetInventoryDataKeepsFamily(t *testing.T) {
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
			Mpn:      "bt2000",
			FamilyID: "family-1",
		},
		{
			Name:     "a-corp/eagle/bt2000-variant",
			FamilyID: "family-1",
			Versions: []model.FoundVersion{{IndexVersion: &model.IndexVersion{TMID: "a-corp/eagle/bt2000-variant/v1.0.0-20240108140117-243d1b462ccd.tm.json"}}},
		},
	}

	inventoryEntries := mapper.GetInventoryData(entries)

	if assert.Len(t, inventoryEntries, 2) {
		assert.Equal(t, "a-corp/eagle/bt2000", inventoryEntries[0].TmName)
		assert.Equal(t, "family-1", *inventoryEntries[0].Family)
		assert.Equal(t, "a-corp/eagle/bt2000-variant", inventoryEntries[1].TmName)
		assert.Equal(t, "family-1", *inventoryEntries[1].Family)
	}
}

func TestMapper_GetInventoryEntryOmitsEmptyFamily(t *testing.T) {
	mapper := NewMapper(context.Background())
	assert.Nil(t, mapper.GetInventoryEntry(model.FoundEntry{}).Family)
}
