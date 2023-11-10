package validation

import (
	"github.com/stretchr/testify/assert"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"testing"
)

func TestValidateAsTM(t *testing.T) {
	_, raw, err := internal.ReadRequiredFile("../../test/data/validation/omnilamp.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../test/data/validation/omnilamp-broken.json")
	assert.NoError(t, err)
	err = ValidateAsTM(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "$.properties.status.readOnly")

}
func TestValidateAsModbus(t *testing.T) {
	_, raw, err := internal.ReadRequiredFile("../../test/data/validation/omnilamp.json")
	assert.NoError(t, err)
	v, err := ValidateAsModbus(raw)
	assert.False(t, v)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../test/data/validation/modbus-senseall.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw)
	assert.True(t, v)
	assert.NoError(t, err)

	_, raw, err = internal.ReadRequiredFile("../../test/data/validation/modbus-senseall-broken.json")
	assert.NoError(t, err)
	v, err = ValidateAsModbus(raw)
	assert.True(t, v)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "$.properties.SERIAL_NUMBER.forms.0.modbus:zeroBasedAddressing")

}
