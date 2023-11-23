package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
)

func TestValidateAsTM(t *testing.T) {
	_, raw, err := internal.ReadRequiredFile("../../../test/data/validate/omnilamp.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../../test/data/validate/omnilamp-broken.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "$.properties.status.readOnly")

}
func TestValidateAsModbus(t *testing.T) {
	_, raw, err := internal.ReadRequiredFile("../../../test/data/validate/omnilamp.json")
	assert.NoError(t, err)
	v, err := ValidateAsModbus(raw)
	assert.False(t, v)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../../test/data/validate/modbus-senseall.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw)
	assert.True(t, v)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../../test/data/validate/modbus-senseall-broken.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw)
	assert.True(t, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "$.properties.SERIAL_NUMBER.forms.0.modbus:zeroBasedAddressing")

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
	tm, err := ValidateAsTmcImportable([]byte(tmFile))
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
	tm, err = ValidateAsTmcImportable([]byte(tmFile))
	assert.Error(t, err)
	assert.ErrorContains(t, err, "$.schema:author.schema:name")

	tmFile = `{
  "@context": [{
    "schema":"https://schema.org/"
  }],
  "schema:mpn": "sense&all",
  "schema:author": {
    "schema:name": "omnicorp"
  }
}`
	_, err = ValidateAsTmcImportable([]byte(tmFile))
	assert.Error(t, err)
	assert.ErrorContains(t, err, "schema:manufacturer at \"$\"")
}
