package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchResult_Filter(t *testing.T) {
	t.Run("filter by name", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut2/man/mpn"})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut2/man/mpn", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut/man/mpn"})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
		}
	})
	t.Run("filter by name with prefix match", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut2", Options: FilterOptions{NameFilterType: PrefixMatch}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut2/man/mpn", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut/man/mpn", Options: FilterOptions{NameFilterType: PrefixMatch}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut/man", Options: FilterOptions{NameFilterType: PrefixMatch}})
		if assert.Len(t, sr.Entries, 2) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut/man/", Options: FilterOptions{NameFilterType: PrefixMatch}})
		if assert.Len(t, sr.Entries, 2) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Name: "aut/man/mpn/sub", Options: FilterOptions{NameFilterType: PrefixMatch}})
		assert.Len(t, sr.Entries, 0)
	})
	t.Run("filter by mpn", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Mpn: []string{"mpn2"}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut/man/mpn2", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Mpn: []string{"mpn", "mpn2", "mpn45"}})
		if assert.Len(t, sr.Entries, 4) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut/man2/mpn", sr.Entries[2].Name)
			assert.Equal(t, "aut2/man/mpn", sr.Entries[3].Name)
		}
	})
	t.Run("filter by manufacturer", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Manufacturer: []string{"man"}})
		if assert.Len(t, sr.Entries, 3) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut2/man/mpn", sr.Entries[2].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Manufacturer: []string{"man", "man2", "mpn45"}})
		assert.Len(t, sr.Entries, 4)
		if assert.Len(t, sr.Entries, 4) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut/man2/mpn", sr.Entries[2].Name)
			assert.Equal(t, "aut2/man/mpn", sr.Entries[3].Name)
		}
	})
	t.Run("filter by author", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Author: []string{"aut2"}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut2/man/mpn", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Author: []string{"aut"}})
		if assert.Len(t, sr.Entries, 3) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut/man2/mpn", sr.Entries[2].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Author: []string{"aut2", "aut"}})
		if assert.Len(t, sr.Entries, 4) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut/man2/mpn", sr.Entries[2].Name)
			assert.Equal(t, "aut2/man/mpn", sr.Entries[3].Name)
		}
	})
	t.Run("filter by protocol", func(t *testing.T) {
		sr := prepareSearchResult()
		sr.Filter(&Filters{Protocol: []string{"https"}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
		}

		sr = prepareSearchResult()
		sr.Filter(&Filters{Protocol: []string{"modbus", "coap", "opcua+tcp"}})
		if assert.Len(t, sr.Entries, 1) {
			assert.Equal(t, "aut2/man/mpn", sr.Entries[0].Name)
		}
	})
	t.Run("filter by author and manufacturer", func(t *testing.T) {
		sr := prepareSearchResult()
		_ = sr.Filter(&Filters{Manufacturer: []string{"man"}, Author: []string{"aut"}})
		if assert.Len(t, sr.Entries, 2) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
		}

		sr = prepareSearchResult()
		_ = sr.Filter(&Filters{Manufacturer: []string{"man"}, Author: []string{"aut2", "aut"}})
		if assert.Len(t, sr.Entries, 3) {
			assert.Equal(t, "aut/man/mpn", sr.Entries[0].Name)
			assert.Equal(t, "aut/man/mpn2", sr.Entries[1].Name)
			assert.Equal(t, "aut2/man/mpn", sr.Entries[2].Name)
		}
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
		r := NewIndexToFoundMapper(EmptySpec.ToFoundSource()).ToSearchResult(*idx)
		sr := &r
		_ = sr.Filter(ToFilters(&author, &manuf, &mpn, nil, nil, nil))
		assert.Len(t, sr.Entries, 1)

		author = "Aut%hor"
		manuf = "Man-ufacturer"
		mpn = "M&pN"
		_ = sr.Filter(ToFilters(&author, &manuf, &mpn, nil, nil, nil))
		assert.Len(t, sr.Entries, 1)
	})
}

func prepareSearchResult() *SearchResult {
	idx := prepareIndex()
	sr := NewIndexToFoundMapper(EmptySpec.ToFoundSource()).ToSearchResult(*idx)
	return &sr
}
