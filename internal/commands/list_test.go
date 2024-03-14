package commands

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes/mocks"
)

func TestListCommand_List(t *testing.T) {
	t.Run("merged", func(t *testing.T) {
		r1 := mocks.NewRemote(t)
		r2 := mocks.NewRemote(t)
		remotes.MockRemotesAll(t, func() ([]remotes.Remote, error) {
			return []remotes.Remote{r1, r2}, nil
		})
		r1.On("List", &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
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
		r2.On("List", &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
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

		res, err, _ := List(model.EmptySpec, &model.SearchParams{Query: "omnicorp"})

		assert.NoError(t, err)
		assert.Len(t, res.Entries, 3)
		assert.Equal(t, "omnicorp/actall", res.Entries[0].Name)
		assert.Equal(t, "omnicorp/lightall", res.Entries[1].Name)
		assert.Equal(t, "omnicorp/senseall", res.Entries[2].Name)
	})
	t.Run("one error", func(t *testing.T) {
		r1 := mocks.NewRemote(t)
		r2 := mocks.NewRemote(t)
		r2.On("Spec").Return(model.NewRemoteSpec("r2"))
		remotes.MockRemotesAll(t, func() ([]remotes.Remote, error) {
			return []remotes.Remote{r1, r2}, nil
		})
		r1.On("List", &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{
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
		r2.On("List", &model.SearchParams{Query: "omnicorp"}).Return(model.SearchResult{}, errors.New("unexpected error"))

		res, err, errs := List(model.EmptySpec, &model.SearchParams{Query: "omnicorp"})

		assert.NoError(t, err)
		if assert.Len(t, errs, 1) {
			assert.ErrorContains(t, errs[0], "unexpected error")
		}
		assert.Len(t, res.Entries, 2)
		assert.Equal(t, "omnicorp/lightall", res.Entries[0].Name)
		assert.Equal(t, "omnicorp/senseall", res.Entries[1].Name)
	})

}
