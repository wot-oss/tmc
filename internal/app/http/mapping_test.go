package http

import (
	"context"
	"encoding/json"
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
		Mpn:        "bt2000",
		FamilyID:   "family-1",
		FamilyName: "Eagle",
	}

	inventoryEntry := mapper.GetInventoryEntry(entry)

	if assert.NotNil(t, inventoryEntry.FamilyId) && assert.NotNil(t, inventoryEntry.FamilyName) {
		assert.Equal(t, "family-1", *inventoryEntry.FamilyId)
		assert.Equal(t, "Eagle", *inventoryEntry.FamilyName)
	}
	raw, err := json.Marshal(inventoryEntry)
	assert.NoError(t, err)
	var document map[string]any
	assert.NoError(t, json.Unmarshal(raw, &document))
	assert.Equal(t, "family-1", document["familyId"])
	assert.Equal(t, "Eagle", document["familyName"])
	assert.NotContains(t, document, "family")
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
			Mpn:        "bt2000",
			FamilyID:   "family-1",
			FamilyName: "Eagle",
		},
		{
			Name:       "a-corp/eagle/bt2000-variant",
			FamilyID:   "family-1",
			FamilyName: "Eagle",
			Versions:   []model.FoundVersion{{IndexVersion: &model.IndexVersion{TMID: "a-corp/eagle/bt2000-variant/v1.0.0-20240108140117-243d1b462ccd.tm.json"}}},
		},
	}

	inventoryEntries := mapper.GetInventoryData(entries)

	if assert.Len(t, inventoryEntries, 2) {
		assert.Equal(t, "a-corp/eagle/bt2000", inventoryEntries[0].TmName)
		assert.Equal(t, "family-1", *inventoryEntries[0].FamilyId)
		assert.Equal(t, "Eagle", *inventoryEntries[0].FamilyName)
		assert.Equal(t, "a-corp/eagle/bt2000-variant", inventoryEntries[1].TmName)
		assert.Equal(t, "family-1", *inventoryEntries[1].FamilyId)
		assert.Equal(t, "Eagle", *inventoryEntries[1].FamilyName)
	}
}

func TestMapper_GetInventoryEntryOmitsEmptyFamily(t *testing.T) {
	mapper := NewMapper(context.Background())
	entry := mapper.GetInventoryEntry(model.FoundEntry{})
	assert.Nil(t, entry.FamilyId)
	assert.Nil(t, entry.FamilyName)
}
