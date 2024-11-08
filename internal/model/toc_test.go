package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIndex_Filter(t *testing.T) {
	t.Run("filter by name", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "man/mpn"})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "aut/man/mpn"})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by name with prefix match", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "man", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "aut/man/mpn", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("man/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "aut/man", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "aut/man/", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Name: "aut/man/mpn/sub", Options: SearchOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, idx.Data, 0)
	})
	t.Run("filter by mpn", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Mpn: []string{"mpn2"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Mpn: []string{"mpn", "mpn2", "mpn45"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by manufacturer", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Manufacturer: []string{"man"}})
		assert.Len(t, idx.Data, 3)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Manufacturer: []string{"man", "man2", "mpn45"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by author", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Author: []string{"man"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man/mpn2"))
		assert.Nil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Author: []string{"aut"}})
		assert.Len(t, idx.Data, 3)
		assert.Nil(t, idx.FindByName("man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Author: []string{"man", "aut"}})
		assert.Len(t, idx.Data, 4)
		assert.NotNil(t, idx.FindByName("man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
	})
	t.Run("filter by protocol", func(t *testing.T) {
		idx := prepareIndex()
		idx.Filter(&SearchParams{Protocol: []string{"https"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))

		idx = prepareIndex()
		idx.Filter(&SearchParams{Protocol: []string{"modbus", "coap", "opcua+tcp"}})
		assert.Len(t, idx.Data, 1)
		assert.NotNil(t, idx.FindByName("man/mpn"))
	})
	t.Run("filter by query", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Query: ""})
		assert.Len(t, idx.Data, 4)

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Query: "z"})
		assert.Len(t, idx.Data, 0)

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Query: "a"})
		assert.Len(t, idx.Data, 4)

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Query: "d1"})
		assert.Len(t, idx.Data, 1)
		assert.Len(t, idx.Data[0].Versions, 2)

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Query: "d5"})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man2/mpn"))
	})
	t.Run("filter by author and manufacturer", func(t *testing.T) {
		idx := prepareIndex()
		_ = idx.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"aut"}})
		assert.Len(t, idx.Data, 2)
		assert.NotNil(t, idx.FindByName("aut/man/mpn2"))
		assert.NotNil(t, idx.FindByName("aut/man/mpn"))
		assert.Nil(t, idx.FindByName("aut/man2/mpn"))

		idx = prepareIndex()
		_ = idx.Filter(&SearchParams{Manufacturer: []string{"man"}, Author: []string{"man", "aut"}})
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
					Versions: []*IndexVersion{
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
		_ = idx.Filter(ToSearchParams(&author, &manuf, &mpn, nil, nil, nil, nil))
		assert.Len(t, idx.Data, 1)

		author = "Aut%hor"
		manuf = "Man-ufacturer"
		mpn = "M&pN"
		_ = idx.Filter(ToSearchParams(&author, &manuf, &mpn, nil, nil, nil, nil))
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
				Versions: []*IndexVersion{
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
						Protocols:   []string{"modbus"},
					},
				},
			},
			{
				Name:         "aut/man/mpn",
				Manufacturer: SchemaManufacturer{"man"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"aut"},
				Versions: []*IndexVersion{
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
						Protocols:   []string{"https"},
					},
				},
			},
			{
				Name:         "aut/man2/mpn",
				Manufacturer: SchemaManufacturer{"man2"},
				Mpn:          "mpn",
				Author:       SchemaAuthor{"aut"},
				Versions: []*IndexVersion{
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
				Versions: []*IndexVersion{
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

func TestIndex_InsertAttachments(t *testing.T) {
	idx := &Index{}
	tmName := "aut/man/mpn"
	id := tmName + "/v1.2.5-20231023121314-abcd12345678.tm.json"
	err := idx.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        []Link{{Rel: "original", HRef: "externalID"}},
		ID:           id,
		Description:  "descr",
	})
	assert.NoError(t, err)
	atts := []Attachment{{
		Name:      "README.md",
		MediaType: "Message/markdown",
	}, {
		Name:      "User Guide.pdf",
		MediaType: "application/pdf",
	}}
	idRef := NewTMIDAttachmentContainerRef(id)
	err = idx.InsertAttachments(idRef, atts...)
	assert.NoError(t, err)
	cnt, _, err := idx.FindAttachmentContainer(idRef)
	assert.NoError(t, err)
	assert.Equal(t, atts, (*cnt).Attachments)

	nameAtts := []Attachment{{
		Name:      "CHANGELOG.md",
		MediaType: "Message/markdown",
	}}
	nameRef := NewTMNameAttachmentContainerRef(tmName)
	err = idx.InsertAttachments(nameRef, nameAtts...)
	assert.NoError(t, err)
	cnt, _, err = idx.FindAttachmentContainer(nameRef)
	assert.NoError(t, err)
	assert.Equal(t, nameAtts, (*cnt).Attachments)

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
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(idx.Data))
	assert.Equal(t, "aut/man/mpn", idx.Data[0].Name)
	assert.Equal(t, 1, len(idx.Data[0].Versions))
	err = idx.InsertAttachments(NewTMIDAttachmentContainerRef("aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json"), Attachment{Name: "README.md", MediaType: "Message/markdown"}, Attachment{Name: "User Guide.pdf", MediaType: "application/pdf"})
	assert.NoError(t, err)
	assert.Equal(t, &IndexVersion{
		Description: "descr",
		Version: Version{
			Model: "1.2.5",
		},
		Links:               map[string]string{"content": "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json"},
		TMID:                "aut/man/mpn/v1.2.5-20231023121314-abcd12345678.tm.json",
		Digest:              "abcd12345678",
		TimeStamp:           "20231023121314",
		ExternalID:          "externalID",
		AttachmentContainer: AttachmentContainer{[]Attachment{{Name: "README.md", MediaType: "Message/markdown"}, {Name: "User Guide.pdf", MediaType: "application/pdf"}}},
	}, idx.Data[0].Versions[0])

	err = idx.Insert(&ThingModel{
		Manufacturer: SchemaManufacturer{Name: "man"},
		Mpn:          "mpn",
		Author:       SchemaAuthor{Name: "aut"},
		Links:        nil,
		ID:           "aut/man/mpn/v1.2.6-20231024121314-abcd12345690.tm.json",
		Description:  "descr",
	})
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
	})
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

func TestIndex_IsEmpty(t *testing.T) {
	idx := &Index{
		Meta: IndexMeta{
			Created: time.Now(),
		},
	}

	// nil Data
	idx.Data = nil
	assert.True(t, idx.IsEmpty())

	// empty slice Data
	idx.Data = []*IndexEntry{}
	assert.True(t, idx.IsEmpty())

	// non-empty Data
	idx.Data = []*IndexEntry{{Name: "some entry"}}
	assert.False(t, idx.IsEmpty())
}

func TestIndex_Sort(t *testing.T) {
	idx := &Index{
		Meta: IndexMeta{
			Created: time.Now(),
		},
	}

	// nil Data
	idx.Data = nil
	assert.NotPanics(t, func() { idx.Sort() })

	// empty slice Data
	idx.Data = []*IndexEntry{}
	assert.NotPanics(t, func() { idx.Sort() })

	// non-empty Data
	idxEntry1 := &IndexEntry{
		Name: "z/y/x",
		Versions: []*IndexVersion{
			{TMID: "z/y/x/v0.1.0-20240606131725-1bbbbbbbbbbb.tm.json", Version: Version{Model: "0.1.0"}},
			{TMID: "z/y/x/v0.11.0-20240606131725-1aaaaaaaaaaa.tm.json", Version: Version{Model: "0.11.0"}},
			{TMID: "z/y/x/v0.2.1-20240606131725-1ccccccccccc.tm.json", Version: Version{Model: "0.2.1"}},
		},
	}

	idxEntry2 := &IndexEntry{
		Name: "a/b/c",
		Versions: []*IndexVersion{
			{TMID: "a/b/c/v0.0.0-20240606131725-1aaaaaaaaaaa.tm.json", Version: Version{Model: "0.0.0"}},
			{TMID: "a/b/c/v0.0.0-20270730131725-1aaaaaaaaaaa.tm.json", Version: Version{Model: "0.0.0"}},
			{TMID: "a/b/c/v0.0.0-20240606131725-1bbbbbbbbbbb.tm.json", Version: Version{Model: "0.0.0"}},
		},
	}

	idx.Data = []*IndexEntry{
		idxEntry1, idxEntry2,
	}

	expIdxData := []*IndexEntry{
		{
			Name:     idxEntry2.Name,
			Versions: []*IndexVersion{idxEntry2.Versions[1], idxEntry2.Versions[2], idxEntry2.Versions[0]},
		},
		{
			Name:     idxEntry1.Name,
			Versions: []*IndexVersion{idxEntry1.Versions[1], idxEntry1.Versions[2], idxEntry1.Versions[0]},
		},
	}

	idx.Sort()

	assert.Equal(t, expIdxData, idx.Data)
}
