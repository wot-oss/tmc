package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFetchName(t *testing.T) {
	tests := []struct {
		in      string
		expErr  bool
		expName string
		expSV   string
	}{
		{"", true, "", ""},
		{"manufacturer", true, "", ""},
		{"manufacturer\\mpn", true, "", ""},
		{"manu-facturer/mpn", true, "", ""},
		{"manufacturer/mpn:1.2.3", true, "", ""},
		{"manufacturer/mpn:1.2.", true, "", ""},
		{"manufacturer/mpn:1.2", true, "", "1.2"},
		{"manufacturer/mpn:v1.2.3", true, "", "v1.2.3"},
		{"manufacturer/mpn:43748209adcb", true, "", ""},
		{"author/manu-facturer/mpn:1.2.3", false, "author/manu-facturer/mpn", "1.2.3"},
		{"author/manufacturer/mpn:v1.2.3", false, "author/manufacturer/mpn", "v1.2.3"},
		{"author/manufacturer/mpn/folder/structure:1.2.3", false, "author/manufacturer/mpn/folder/structure", "1.2.3"},
		{"author/manufacturer/mpn/folder/structure:v1.2.3-alpha1", false, "author/manufacturer/mpn/folder/structure", "v1.2.3-alpha1"},
	}

	for _, test := range tests {
		out, err := ParseFetchName(test.in)
		if test.expErr {
			assert.Error(t, err, "Want: error in ParseFetchName(%s). Got: nil", test.in)
			assert.ErrorIs(t, err, ErrInvalidFetchName)
		} else {
			assert.NoError(t, err, "Want: no error in ParseFetchName(%s). Got: %v", test.in, err)
			exp := FetchName{test.expName, test.expSV}
			assert.Equal(t, exp, out, "Want: ParseFetchName(%s) = %v. Got: %v", test.in, exp, out)
		}
	}
}
