package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTOC_Filter(t *testing.T) {
	t.Run("filter by name", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Name: "man/mpn"})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("man/mpn"))
		assert.Nil(t, toc.findByName("aut/man/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Name: "aut/man/mpn"})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.Nil(t, toc.findByName("man/mpn"))
	})
	t.Run("filter by name with prefix match", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Name: "man", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("man/mpn"))
		assert.Nil(t, toc.findByName("aut/man/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Name: "aut/man/", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, toc.Data, 2)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.Nil(t, toc.findByName("man/mpn"))
	})
	t.Run("filter by mpn", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Mpn: []string{"mpn2"}})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.Nil(t, toc.findByName("aut/man/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Mpn: []string{"mpn", "mpn2", "mpn45"}})
		assert.Len(t, toc.Data, 4)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("man/mpn"))
	})
	t.Run("filter by manufacturer", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Manufacturer: []string{"man"}})
		assert.Len(t, toc.Data, 3)
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.Nil(t, toc.findByName("aut/man2/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Manufacturer: []string{"man", "man2", "mpn45"}})
		assert.Len(t, toc.Data, 4)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man2/mpn"))
		assert.NotNil(t, toc.findByName("man/mpn"))
	})
	t.Run("filter by author", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Author: []string{"man"}})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("man/mpn"))
		assert.Nil(t, toc.findByName("aut/man/mpn2"))
		assert.Nil(t, toc.findByName("aut/man/mpn"))
		assert.Nil(t, toc.findByName("aut/man2/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Author: []string{"aut"}})
		assert.Len(t, toc.Data, 3)
		assert.Nil(t, toc.findByName("man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man2/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Author: []string{"man", "aut"}})
		assert.Len(t, toc.Data, 4)
		assert.NotNil(t, toc.findByName("man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man2/mpn"))
	})
	t.Run("filter by query", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Query: ""})
		assert.Len(t, toc.Data, 4)

		toc = prepareToc()
		toc.Filter(&SearchParams{Query: "z"})
		assert.Len(t, toc.Data, 0)

		toc = prepareToc()
		toc.Filter(&SearchParams{Query: "a"})
		assert.Len(t, toc.Data, 4)

		toc = prepareToc()
		toc.Filter(&SearchParams{Query: "d1"})
		assert.Len(t, toc.Data, 1)
		assert.Len(t, toc.Data[0].Versions, 2)

		toc = prepareToc()
		toc.Filter(&SearchParams{Query: "d5"})
		assert.Len(t, toc.Data, 2)
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man2/mpn"))
	})
	t.Run("filter by author and manufacturer", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"aut"}})
		assert.Len(t, toc.Data, 2)
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.Nil(t, toc.findByName("aut/man2/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"man", "aut"}})
		assert.Len(t, toc.Data, 3)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("aut/man/mpn2"))
		assert.NotNil(t, toc.findByName("man/mpn"))
	})
	t.Run("filter by externalID", func(t *testing.T) {
		toc := prepareToc()
		toc.Filter(&SearchParams{ExternalID: []string{"externalID"}, Author: []string{"aut"}})
		assert.Len(t, toc.Data, 0)

		toc = prepareToc()
		toc.Filter(&SearchParams{ExternalID: []string{"externalID"}})
		assert.Len(t, toc.Data, 1)
		assert.NotNil(t, toc.findByName("man/mpn"))

		toc = prepareToc()
		toc.Filter(&SearchParams{ExternalID: []string{"externalID", "externalID2"}})
		assert.Len(t, toc.Data, 2)
		assert.NotNil(t, toc.findByName("aut/man/mpn"))
		assert.NotNil(t, toc.findByName("man/mpn"))
	})
}

func prepareToc() *TOC {
	toc := &TOC{
		Meta: TOCMeta{},
		Data: []*TOCEntry{
			{
				Name:         "man/mpn",
				Manufacturer: SchemaManufacturer{"man"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"man"},
				Versions: []TOCVersion{
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
				Versions: []TOCVersion{
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
				Versions: []TOCVersion{
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
				Versions: []TOCVersion{
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
	return toc
}

func TestTOC_Insert(t *testing.T) {
	toc := &TOC{}

	err := toc.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        []Link{{Rel: "original", HRef: "externalID"}},
		ID:           "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json",
		Description:  "descr",
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(toc.Data))
	assert.Equal(t, "aut/man/mpn", toc.Data[0].Name)
	assert.Equal(t, 1, len(toc.Data[0].Versions))
	assert.Equal(t, TOCVersion{
		Description: "descr",
		Version: Version{
			Model: "1.2.5",
		},
		Links:      map[string]string{"content": "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json"},
		TMID:       "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json",
		Digest:     "abcd12345678",
		TimeStamp:  "20231023121314",
		ExternalID: "externalID",
	}, toc.Data[0].Versions[0])

	err = toc.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        nil,
		ID:           "aut/man/mpn/v1.2.6-20231024121314-abcd12345690.tm.json",
		Description:  "descr",
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(toc.Data))
	assert.Equal(t, 2, len(toc.Data[0].Versions))

	err = toc.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        nil,
		ID:           "aut/man/mpn/opt/v1.2.6-20231024121314-abcd12345690.tm.json",
		Description:  "descr",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(toc.Data))
	assert.Equal(t, "aut/man/mpn/opt", toc.Data[1].Name)
	assert.Equal(t, 2, len(toc.Data[0].Versions))
	assert.Equal(t, 1, len(toc.Data[1].Versions))
}
