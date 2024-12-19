package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos/mocks"
	rMocks "github.com/wot-oss/tmc/internal/testutils/reposmocks"
)

func TestListCommand_List(t *testing.T) {
	params := &model.Filters{Name: "omnicorp"}
	t.Run("merged", func(t *testing.T) {
		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("List", mock.Anything, params).Return(model.SearchResult{
			Entries: []model.FoundEntry{
				{
					Name:         "omnicorp/senseall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "senseall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
					FoundIn:      model.FoundSource{RepoName: "r1"},
				},
				{
					Name:         "omnicorp/lightall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "lightall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
					FoundIn:      model.FoundSource{RepoName: "r1"},
				},
			},
		}, nil)
		r2.On("List", mock.Anything, params).Return(model.SearchResult{
			Entries: []model.FoundEntry{
				{
					Name:         "omnicorp/senseall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "senseall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
					FoundIn:      model.FoundSource{RepoName: "r2"},
				},
				{
					Name:         "omnicorp/actall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "actall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
					FoundIn:      model.FoundSource{RepoName: "r2"},
				},
			},
		}, nil)

		res, err, _ := List(context.Background(), model.EmptySpec, params)

		assert.NoError(t, err)
		if assert.Len(t, res.Entries, 4) {
			assert.Equal(t, "omnicorp/actall", res.Entries[0].Name)
			assert.Equal(t, "r2", res.Entries[0].FoundIn.RepoName)
			assert.Equal(t, "omnicorp/lightall", res.Entries[1].Name)
			assert.Equal(t, "omnicorp/senseall", res.Entries[2].Name)
			assert.Equal(t, "r1", res.Entries[2].FoundIn.RepoName)
			assert.Equal(t, "omnicorp/senseall", res.Entries[3].Name)
			assert.Equal(t, "r2", res.Entries[3].FoundIn.RepoName)
		}
	})
	t.Run("one error", func(t *testing.T) {
		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		r2.On("Spec").Return(model.NewRepoSpec("r2"))
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("List", mock.Anything, params).Return(model.SearchResult{
			Entries: []model.FoundEntry{
				{
					Name:         "omnicorp/senseall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "senseall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
				},
				{
					Name:         "omnicorp/lightall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "lightall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
				},
			},
		}, nil)
		r2.On("List", mock.Anything, params).Return(model.SearchResult{}, errors.New("unexpected error"))

		res, err, errs := List(context.Background(), model.EmptySpec, params)

		assert.NoError(t, err)
		if assert.Len(t, errs, 1) {
			assert.ErrorContains(t, errs[0], "unexpected error")
		}
		assert.Len(t, res.Entries, 2)
		assert.Equal(t, "omnicorp/lightall", res.Entries[0].Name)
		assert.Equal(t, "omnicorp/senseall", res.Entries[1].Name)
	})

}
