package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

func resetSearchFlags(flags *FilterFlags) {
	flags.FilterAuthor = ""
	flags.FilterManufacturer = ""
	flags.FilterMpn = ""
	flags.FilterExternalID = ""
	flags.Search = ""
}

func TestFilterFlagsSet(t *testing.T) {
	underTest := FilterFlags{}

	resetSearchFlags(&underTest)
	underTest.FilterAuthor = "some value"
	assert.True(t, underTest.IsSet())

	resetSearchFlags(&underTest)
	underTest.FilterManufacturer = "some value"
	assert.True(t, underTest.IsSet())

	resetSearchFlags(&underTest)
	underTest.FilterMpn = "some value"
	assert.True(t, underTest.IsSet())

	resetSearchFlags(&underTest)
	underTest.FilterExternalID = "some value"
	assert.True(t, underTest.IsSet())

	resetSearchFlags(&underTest)
	underTest.Search = "some value"
	assert.True(t, underTest.IsSet())

	resetSearchFlags(&underTest)
	assert.False(t, underTest.IsSet())
}

func TestConvertSearchParams(t *testing.T) {

	// given: no filter params set via CLI flags
	flags := FilterFlags{}
	// when: converting to SearchParams
	params := CreateSearchParamsFromCLI(flags, "")
	// then: SearchParams are undefined
	assert.Nil(t, params)

	// given: filter params are set with single values
	resetSearchFlags(&flags)
	flags.FilterAuthor = "some author"
	flags.FilterManufacturer = "some manufacturer"
	flags.FilterMpn = "some mpn"
	flags.FilterExternalID = "some externalID"
	flags.Search = "some term"
	name := "omni-corp/omni"
	// when: converting to SearchParams
	params = CreateSearchParamsFromCLI(flags, name)
	// then: the filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, []string{flags.FilterAuthor}, params.Author)
	assert.Equal(t, []string{flags.FilterManufacturer}, params.Manufacturer)
	assert.Equal(t, []string{flags.FilterMpn}, params.Mpn)
	assert.Equal(t, []string{flags.FilterExternalID}, params.ExternalID)
	assert.Equal(t, name, params.Name)
	assert.Equal(t, model.PrefixMatch, params.Options.NameFilterType)
	assert.Equal(t, flags.Search, params.Query)

	// given: filter params are set with multiple comma-separated values
	resetSearchFlags(&flags)
	flags.FilterAuthor = "some author 1,some author 2"
	flags.FilterManufacturer = "some manufacturer 1,some manufacturer 2"
	flags.FilterMpn = "some mpn 1,some mpn 2,some mpn 3"
	flags.FilterExternalID = "some externalID 1,some external ID 2"
	flags.Search = "some term"
	// when: converting to SearchParams
	params = CreateSearchParamsFromCLI(flags, "")
	// then: the multiple filter values are converted correctly
	assert.NotNil(t, params)
	assert.Equal(t, strings.Split(flags.FilterAuthor, ","), params.Author)
	assert.Equal(t, strings.Split(flags.FilterManufacturer, ","), params.Manufacturer)
	assert.Equal(t, strings.Split(flags.FilterMpn, ","), params.Mpn)
	assert.Equal(t, strings.Split(flags.FilterExternalID, ","), params.ExternalID)
	assert.Equal(t, flags.Search, params.Query)
}
