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
	t.Run("merged", func(t *testing.T) {
		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("List", mock.Anything, &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
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
		r2.On("List", mock.Anything, &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
			Entries: []model.FoundEntry{
				{
					Name:         "omnicorp/senseall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "senseall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
				},
				{
					Name:         "omnicorp/actall",
					Manufacturer: model.SchemaManufacturer{Name: "omnicorp"},
					Mpn:          "actall",
					Author:       model.SchemaAuthor{Name: "omnicorp"},
				},
			},
		}, nil)

		res, err, _ := List(context.Background(), model.EmptySpec, &model.SearchParams{Query: "omnicorp"})

		assert.NoError(t, err)
		assert.Len(t, res.Entries, 3)
		assert.Equal(t, "omnicorp/actall", res.Entries[0].Name)
		assert.Equal(t, "omnicorp/lightall", res.Entries[1].Name)
		assert.Equal(t, "omnicorp/senseall", res.Entries[2].Name)
	})
	t.Run("one error", func(t *testing.T) {
		r1 := mocks.NewRepo(t)
		r2 := mocks.NewRepo(t)
		r2.On("Spec").Return(model.NewRepoSpec("r2"))
		rMocks.MockReposAll(t, rMocks.CreateMockAllFunction(nil, r1, r2))
		r1.On("List", mock.Anything, &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
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
		r2.On("List", mock.Anything, &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{}, errors.New("unexpected error"))

		res, err, errs := List(context.Background(), model.EmptySpec, &model.SearchParams{Query: "omnicorp"})

		assert.NoError(t, err)
		if assert.Len(t, errs, 1) {
			assert.ErrorContains(t, errs[0], "unexpected error")
		}
		assert.Len(t, res.Entries, 2)
		assert.Equal(t, "omnicorp/lightall", res.Entries[0].Name)
		assert.Equal(t, "omnicorp/senseall", res.Entries[1].Name)
	})

}
