package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wot-oss/tmc/internal/app/http/server"
)

func TestFamilyMetadataMappings(t *testing.T) {
	entry := &IndexEntry{Name: "aut/man/source", FamilyID: "family-1", FamilyName: "Eagle"}
	raw, err := json.Marshal(entry)
	require.NoError(t, err)
	var stored IndexEntry
	require.NoError(t, json.Unmarshal(raw, &stored))
	found := NewIndexToFoundMapper(FoundSource{}).ToFoundEntry(&stored)
	assert.Equal(t, "family-1", found.FamilyID)
	assert.Equal(t, "Eagle", found.FamilyName)

	var inventory server.InventoryEntry
	require.NoError(t, json.Unmarshal([]byte(`{"familyId":"family-1","familyName":"Eagle"}`), &inventory))
	mapped := NewInventoryResponseToSearchResultMapper(FoundSource{}, nil).ToFoundEntry(inventory)
	assert.Equal(t, "family-1", mapped.FamilyID)
	assert.Equal(t, "Eagle", mapped.FamilyName)
}

func TestLegacyFamilyIDRemainsReadable(t *testing.T) {
	var stored IndexEntry
	require.NoError(t, json.Unmarshal([]byte(`{"name":"aut/man/source","family":"family-1"}`), &stored))
	assert.Equal(t, "family-1", stored.FamilyID)
	assert.Empty(t, stored.FamilyName)
}
