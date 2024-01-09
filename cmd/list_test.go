package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func resetSearchFlags() {
	flagFilterAuthor = ""
	flagFilterMpn = ""
	flagFilterManufacturer = ""
	flagFilterExternalID = ""
	flagSearch = ""
}

func TestHasSearchParams(t *testing.T) {
	resetSearchFlags()
	flagFilterAuthor = "some value"
	res := hasSearchParamsSet()
	assert.True(t, res)

	resetSearchFlags()
	flagFilterManufacturer = "some value"
	res = hasSearchParamsSet()
	assert.True(t, res)

	resetSearchFlags()
	flagFilterMpn = "some value"
	res = hasSearchParamsSet()
	assert.True(t, res)

	resetSearchFlags()
	flagFilterExternalID = "some value"
	res = hasSearchParamsSet()
	assert.True(t, res)

	resetSearchFlags()
	flagSearch = "some value"
	res = hasSearchParamsSet()
	assert.True(t, res)

	resetSearchFlags()
	res = hasSearchParamsSet()
	assert.False(t, res)
}

func TestConvertSearchParams(t *testing.T) {
	// given: no filter params set via CLI flags
	resetSearchFlags()
	// when: converting to SearchParams
	params := convertSearchParams()
	// then: SearchParams are undefined
	assert.Nil(t, params)

	// given: filter params are set with single values
	resetSearchFlags()
	flagFilterAuthor = "some author"
	flagFilterManufacturer = "some manufacturer"
	flagFilterMpn = "some mpn"
	flagFilterExternalID = "some externalID"
	flagSearch = "some term"
	// when: converting to SearchParams
	params = convertSearchParams()
	// then: the filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, []string{"some author"}, params.Author)
	assert.Equal(t, []string{"some manufacturer"}, params.Manufacturer)
	assert.Equal(t, []string{"some mpn"}, params.Mpn)
	assert.Equal(t, []string{"some externalID"}, params.ExternalID)
	assert.Equal(t, "some term", params.Query)

	// given: filter params are set with multiple comma-separated values
	resetSearchFlags()
	flagFilterAuthor = "some author 1,some author 2"
	flagFilterManufacturer = "some manufacturer 1,some manufacturer 2"
	flagFilterMpn = "some mpn 1,some mpn 2,some mpn 3"
	flagFilterExternalID = "some externalID 1,some external ID 2"
	flagSearch = "some term"
	// when: converting to SearchParams
	params = convertSearchParams()
	// then: the multiple filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, []string{"some author 1", "some author 2"}, params.Author)
	assert.Equal(t, []string{"some manufacturer 1", "some manufacturer 2"}, params.Manufacturer)
	assert.Equal(t, []string{"some mpn 1", "some mpn 2", "some mpn 3"}, params.Mpn)
	assert.Equal(t, []string{"some externalID 1", "some external ID 2"}, params.ExternalID)
	assert.Equal(t, "some term", params.Query)
}
