package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndex_Filter(t *testing.T) {
	t.Run("filter by name", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Name: "man/mpn"})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Name: "aut/man/mpn"})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by name with prefix match", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Name: "man", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Name: "aut/man/mpn", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("man/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Name: "aut/man", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Name: "aut/man/", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Name: "aut/man/mpn/sub", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 0)
	})
	t.Run("filter by mpn", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Mpn: []string{"mpn2"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Mpn: []string{"mpn", "mpn2", "mpn45"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by manufacturer", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Manufacturer: []string{"man"}})
		assert.Len(t, idx.Data, 3)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Manufacturer: []string{"man", "man2", "mpn45"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by author", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Author: []string{"man"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Author: []string{"aut"}})
		assert.Len(t, idx.Data, 3)
		assert.Nil(t, idx.FindByName("man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Author: []string{"man", "aut"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
	})
	t.Run("filter by query", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Query: ""})
		assert.Len(t, idx.Data, 4)

		idx = prepareIndex()
		idx.Filter(&SearchParams{Query: "z"})
		assert.Len(t, idx.Data, 0)

		idx = prepareIndex()
		idx.Filter(&SearchParams{Query: "a"})
		assert.Len(t, idx.Data, 4)

		idx = prepareIndex()
		idx.Filter(&SearchParams{Query: "d1"})
		assert.Len(t, idx.Data, 1)
		assert.Len(t, idx.Data[0].Versions, 2)

		idx = prepareIndex()
		idx.Filter(&SearchParams{Query: "d5"})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
	})
	t.Run("filter by author and manufacturer", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"aut"}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"man", "aut"}})
		assert.Len(t, idx.Data, 3)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by sanitized key fields", func(t *testing.T) {
		idx := &Index{
			Meta: IndexMeta{},
			Data: []*IndexEntry{
				{
					Name:         "aut-hor/man-ufacturer/m-pn",
					Manufacturer: SchemaManufacturer{"Man&ufacturer"},
					Mpn:          "M/PN",
					Author:       SchemaAuthor{"aut^hor"},
					Versions: []IndexVersion{
						{
							Description: "d2",
							Version:     Version{"1.0.0"},
							TMID:        "aut/man/mpn/v1.0.0-20231023121314-abcd12345680.tm.json",
							Digest:      "abcd12345680",
							TimeStamp:   "20231023121314",
						},
					},
				},
			},
		}
		author := "aut^hor"
		manuf := "Man&ufacturer"
		mpn := "M/PN"
		idx.Filter(ToSearchParams(&author, &manuf, &mpn, nil, nil, nil))
		assert.Len(t, idx.Data, 1)

		author = "Aut%hor"
		manuf = "Man-ufacturer"
		mpn = "M&pN"
		idx.Filter(ToSearchParams(&author, &manuf, &mpn, nil, nil, nil))
		assert.Len(t, idx.Data, 1)
	})
}

func prepareIndex() *Index {
	idx := &Index{
		Meta: IndexMeta{},
		Data: []*IndexEntry{
			{
				Name:         "man/mpn",
				Manufacturer: SchemaManufacturer{"man"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"man"},
				Versions: []IndexVersion{
					{
						Description: "d1",
						Version:     Version{"1.0.0"},
						TMID:        "man/mpn/v1.0.0-20231023121314-abcd12345678.tm.json",
						Digest:      "abcd12345678",
						TimeStamp:   "20231023121314",
					},
					{
						Description: "d1",
						Version:     Version{"1.0.1"},
						TMID:        "man/mpn/v1.0.1-20231024121314-abcd12345679.tm.json",
						ExternalID:  "externalID",
						Digest:      "abcd12345679",
						TimeStamp:   "20231024121314",
					},
				},
			},
			{
				Name:         "aut/man/mpn",
				Manufacturer: SchemaManufacturer{"man"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"aut"},
				Versions: []IndexVersion{
					{
						Description: "d2",
						Version:     Version{"1.0.0"},
						TMID:        "aut/man/mpn/v1.0.0-20231023121314-abcd12345680.tm.json",
						Digest:      "abcd12345680",
						TimeStamp:   "20231023121314",
					},
					{
						Description: "d3",
						Version:     Version{"1.0.1"},
						TMID:        "aut/man/mpn/v1.0.1-20231024121314-abcd12345681.tm.json",
						ExternalID:  "externalID2",
						Digest:      "abcd12345681",
						TimeStamp:   "20231024121314",
					},
				},
			},
			{
				Name:         "aut/man2/mpn",
				Manufacturer: SchemaManufacturer{"man2"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"aut"},
				Versions: []IndexVersion{
					{
						Description: "d4",
						Version:     Version{"1.0.0"},
						TMID:        "aut/man2/mpn/v1.0.0-20231023121314-abcd12345680.tm.json",
						Digest:      "abcd12345680",
						TimeStamp:   "20231023121314",
					},
					{
						Description: "d5",
						Version:     Version{"1.0.1"},
						TMID:        "aut/man2/mpn/v1.0.1-20231024121314-abcd12345681.tm.json",
						Digest:      "abcd12345681",
						TimeStamp:   "20231024121314",
					},
				},
			},
			{
				Name:         "aut/man/mpn2",
				Manufacturer: SchemaManufacturer{"man"},
				Mpn:          "mpn2",
				Author:       SchemaAuthor{"aut"},
				Versions: []IndexVersion{
					{
						Description: "d5",
						Version:     Version{"1.0.0"},
						TMID:        "aut/man/mpn2/v1.0.0-20231023121314-abcd12345680.tm.json",
						Digest:      "abcd12345680",
						TimeStamp:   "20231023121314",
					},
					{
						Description: "d6",
						Version:     Version{"1.0.1"},
						TMID:        "aut/man/mpn2/v1.0.1-20231024121314-abcd12345681.tm.json",
						Digest:      "abcd12345681",
						TimeStamp:   "20231024121314",
					},
				},
			},
		},
	}
	return idx
}
func TestIndex_Insert(t *testing.T) {
	idx := &Index{}

	err := idx.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        []Link{{Rel: "original", HRef: "externalID"}},
		ID:           "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json",
		Description:  "descr",
	}, []string{"README.md", "User Guide.pdf"})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(idx.Data))
	assert.Equal(t, "aut/man/mpn", idx.Data[0].Name)
	assert.Equal(t, 1, len(idx.Data[0].Versions))
	assert.Equal(t, IndexVersion{
		Description: "descr",
		Version: Version{
			Model: "1.2.5",
		},
		Links:       map[string]string{"content": "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json"},
		TMID:        "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json",
		Digest:      "abcd12345678",
		TimeStamp:   "20231023121314",
		ExternalID:  "externalID",
		Attachments: []Attachment{{Name: "README.md"}, {Name: "User Guide.pdf"}},
	}, idx.Data[0].Versions[0])

	err = idx.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        nil,
		ID:           "aut/man/mpn/v1.2.6-20231024121314-abcd12345690.tm.json",
		Description:  "descr",
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(idx.Data))
	assert.Equal(t, 2, len(idx.Data[0].Versions))

	err = idx.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        nil,
		ID:           "aut/man/mpn/opt/v1.2.6-20231024121314-abcd12345690.tm.json",
		Description:  "descr",
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(idx.Data))
	assert.Equal(t, "aut/man/mpn/opt", idx.Data[1].Name)
	assert.Equal(t, 2, len(idx.Data[0].Versions))
	assert.Equal(t, 1, len(idx.Data[1].Versions))
}

func TestIndex_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		expUpdated bool
		expName    string
		expErr     error
	}{
		{
			name:       "invalid id",
			id:         "invalid-id",
			expUpdated: false,
			expName:    "",
			expErr:     ErrInvalidId,
		},
		{
			name:       "non-existing id",
			id:         "aut/man/mpn/opt/v0.0.0-20231024121314-abcd12345690.tm.json",
			expUpdated: false,
			expName:    "",
			expErr:     nil,
		},
		{
			name:       "existing id",
			id:         "aut/man/mpn2/v1.0.0-20231023121314-abcd12345680.tm.json",
			expUpdated: true,
			expName:    "",
			expErr:     nil,
		},
		{
			name:       "last id for a name",
			id:         "man/mpn/v1.0.1-20231024121314-abcd12345679.tm.json",
			expUpdated: true,
			expName:    "man/mpn",
			expErr:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			// prepare an index where one of the names has only one version
			idx := prepareIndex()
			idx.Data[0].Versions = idx.Data[0].Versions[1:]

			updated, name, err := idx.Delete(test.id)

			if test.expErr != nil {
				assert.ErrorIs(t, err, test.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expUpdated, updated)
				assert.Equal(t, test.expName, name)
			}
		})
	}

}
