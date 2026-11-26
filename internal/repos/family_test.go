package repos

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wot-oss/tmc/internal/model"
)

func TestSetFamilyIndexUpdater(t *testing.T) {
	for _, tt := range []struct {
		name        string
		storedName  string
		requestName string
		wantErr     error
	}{
		{"new member", "Eagle", "Eagle", nil},
		{"name an existing unnamed family separately", "", "Eagle", nil},
		{"conflicting name", "Eagle", "Other", model.ErrFamilyNameConflict},
		{"empty name", "Eagle", "", model.ErrFamilyNameRequired},
		{"blank name", "Eagle", " \t ", model.ErrFamilyNameRequired},
	} {
		t.Run(tt.name, func(t *testing.T) {
			member := &model.IndexEntry{Name: "aut/man/source", FamilyID: "family-1", FamilyName: tt.storedName}
			target := &model.IndexEntry{Name: "aut/man/variant"}
			index := &model.Index{Data: []*model.IndexEntry{member, target}}
			_, _, _, err := setFamilyIndexUpdater(target.Name, "family-1", tt.requestName)(context.Background(), index, nil)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, tt.storedName, member.FamilyName)
				assert.Empty(t, target.FamilyID)
				assert.Empty(t, target.FamilyName)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "family-1", target.FamilyID)
				assert.Equal(t, tt.requestName, target.FamilyName)
				assert.Equal(t, tt.requestName, member.FamilyName)
			}
		})
	}
}

func TestSetFamilyIndexUpdaterChecksEveryMemberBeforeChangingIndex(t *testing.T) {
	unnamed := &model.IndexEntry{Name: "aut/man/source", FamilyID: "family-1"}
	conflicting := &model.IndexEntry{Name: "aut/man/other", FamilyID: "family-1", FamilyName: "Original"}
	target := &model.IndexEntry{Name: "aut/man/variant"}
	index := &model.Index{Data: []*model.IndexEntry{unnamed, conflicting, target}}
	_, _, _, err := setFamilyIndexUpdater(target.Name, "family-1", "Different")(context.Background(), index, nil)
	assert.ErrorIs(t, err, model.ErrFamilyNameConflict)
	assert.Empty(t, unnamed.FamilyName)
	assert.Empty(t, target.FamilyID)
	assert.Empty(t, target.FamilyName)
}

func TestSetFamilyIndexUpdaterRejectsEmptyID(t *testing.T) {
	member := &model.IndexEntry{Name: "aut/man/source"}
	index := &model.Index{Data: []*model.IndexEntry{member}}
	_, _, _, err := setFamilyIndexUpdater(member.Name, "", "Eagle")(context.Background(), index, nil)
	assert.ErrorContains(t, err, "family ID must not be empty")
	assert.Empty(t, member.FamilyName)
	assert.Empty(t, member.FamilyID)
}

func TestReindexPreservesFamilyName(t *testing.T) {
	version := &model.IndexVersion{TMID: "aut/man/source/v1.0.0-20260326150433-965fd7c3238b.tm.json"}
	oldIndex := &model.Index{Data: []*model.IndexEntry{{
		Name: "aut/man/source", FamilyID: "family-1", FamilyName: "Eagle",
	}}}
	for _, tt := range []struct {
		name     string
		preserve func(*model.Index, *model.Index)
	}{
		{"file", (&FileRepo{}).preserveFamilies},
		{"s3", (&S3Repo{}).preserveFamilies},
	} {
		t.Run(tt.name, func(t *testing.T) {
			newIndex := &model.Index{Data: []*model.IndexEntry{{
				Name: "aut/man/source", Versions: []*model.IndexVersion{version},
			}}}
			tt.preserve(oldIndex, newIndex)
			assert.Equal(t, "family-1", newIndex.Data[0].FamilyID)
			assert.Equal(t, "Eagle", newIndex.Data[0].FamilyName)
		})
	}
}
