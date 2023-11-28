package validate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
)

func parseJsonFile(name string) ([]byte, any, error) {
	_, raw, err := internal.ReadRequiredFile(name)
	if err != nil {
		return raw, nil, err
	}
	var js any
	err = json.Unmarshal(raw, &js)
	return raw, js, err
}
func parseString(content string) ([]byte, any, error) {
	raw := []byte(content)
	var js any
	err := json.Unmarshal(raw, &js)
	return raw, js, err
}
func TestValidateAsTM(t *testing.T) {
	raw, parsed, err := parseJsonFile("../../../test/data/validate/omnilamp.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw, parsed)
	assert.NoError(t, err)

	raw, parsed, err = parseJsonFile("../../../test/data/validate/omnilamp-broken.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw, parsed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/properties/status/readOnly")

}
func TestValidateAsModbus(t *testing.T) {
	raw, parsed, err := parseJsonFile("../../../test/data/validate/omnilamp.json")
	assert.NoError(t, err)
	v, err := ValidateAsModbus(raw, parsed)
	assert.False(t, v)
	assert.NoError(t, err)

	raw, parsed, err = parseJsonFile("../../../test/data/validate/modbus-senseall.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw, parsed)
	assert.True(t, v)
	assert.NoError(t, err)

	raw, parsed, err = parseJsonFile("../../../test/data/validate/modbus-senseall-broken.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw, parsed)
	assert.True(t, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/properties/SERIAL_NUMBER/forms/0/modbus:zeroBasedAddressing")

}

func TestParseRequiredMetadata(t *testing.T) {
	tmFile := `{
  "@context": [{
    "schema":"https://schema.org/"
  }],
  "schema:manufacturer": {
    "schema:name": "omnicorp GmbH & Co. KG"
  },
  "schema:mpn": "sense&all",
  "schema:author": {
    "schema:name": "omnicorp R&D/research"
  }
}`
	raw, parsed, err := parseString(tmFile)
	assert.NoError(t, err)
	tm, err := ValidateAsTmcImportable(raw, parsed)
	assert.NoError(t, err)
	assert.Equal(t, "omnicorp-GmbH-Co-KG", tm.Manufacturer.Name)
	assert.Equal(t, "omnicorp-R-D-research", tm.Author.Name)
	assert.Equal(t, "sense-all", tm.Mpn)

	tmFile = `{
 "@context": [{
   "schema":"https://schema.org/"
 }],
 "schema:manufacturer": {
   "schema:name": "omnicorp GmbH & Co. KG"
 },
 "schema:mpn": "sense&all",
 "schema:author": {
   "schema:name": " .-"
 }
}`
	raw, parsed, err = parseString(tmFile)
	assert.NoError(t, err)
	_, err = ValidateAsTmcImportable(raw, parsed)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "/schema:author/schema:name")

	tmFile = `{
  "@context": [{
    "schema":"https://schema.org/"
  }],
  "schema:mpn": "sense&all",
  "schema:author": {
    "schema:name": "omnicorp"
  }
}`
	raw, parsed, err = parseString(tmFile)
	assert.NoError(t, err)
	_, err = ValidateAsTmcImportable(raw, parsed)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "missing properties: 'schema:manufacturer'")
}
