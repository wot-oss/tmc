package validate

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/utils"
)

//go:embed tm-json-schema-validation.json
var tmValidationSchema string

//go:embed modbus-old.schema.json
var modbusOldValidationSchema string

//go:embed modbus.schema.json
var modbusValidationSchema string

//go:embed tmc-mandatory.schema.json
var tmcMandatorySchema string

var tmcMandatoryValidator *jsonschema.Schema
var tmValidator *jsonschema.Schema
var modbusOldValidator *jsonschema.Schema
var modbusValidator *jsonschema.Schema

const (
	tmcMandatorySchemaUrl = "resource://tmc-mandatory.schema.json"

	tmSchemaUrl        = "https://raw.githubusercontent.com/w3c/wot-thing-description/main/validation/tm-json-schema-validation.json"
	modbusOldSchemaUrl = "resource://modbus-old.schema.json"
	modbusSchemaUrl    = "resource://modbus.schema.json" // https://w3c.github.io/wot-binding-templates/bindings/protocols/modbus/modbus.schema.json
)

func init() {
	tmcMandatoryValidator = jsonschema.MustCompileString(tmcMandatorySchemaUrl, tmcMandatorySchema)
	tmValidator = jsonschema.MustCompileString(tmSchemaUrl, tmValidationSchema)

	modbusOldCompiler := jsonschema.NewCompiler()
	err := modbusOldCompiler.AddResource(tmSchemaUrl, strings.NewReader(tmValidationSchema))
	if err != nil {
		panic(err)
	}
	err = modbusOldCompiler.AddResource(modbusOldSchemaUrl, strings.NewReader(modbusOldValidationSchema))
	if err != nil {
		panic(err)
	}
	modbusOldValidator = modbusOldCompiler.MustCompile(modbusOldSchemaUrl)

	modbusCompiler := jsonschema.NewCompiler()
	err = modbusCompiler.AddResource(tmSchemaUrl, strings.NewReader(tmValidationSchema))
	if err != nil {
		panic(err)
	}
	err = modbusCompiler.AddResource(modbusSchemaUrl, strings.NewReader(modbusValidationSchema))
	if err != nil {
		panic(err)
	}
	modbusValidator = modbusCompiler.MustCompile(modbusSchemaUrl)

}

func ValidateAsTM(_ []byte, parsed any) error {
	return tmValidator.Validate(parsed)
}

// ValidateAsModbus validates a file against modbus protocol binding json schema, but only if it determines that
// the file purports to describe a modbus device.
// Returns a flag indicating whether validate has been attempted and an error if it was not successful
func ValidateAsModbus(raw []byte, parsed any) (bool, error) {
	if shouldTryModbus(raw) {
		return true, modbusValidator.Validate(parsed)
	}
	if shouldTryOldModbus(raw) {
		return true, modbusOldValidator.Validate(parsed)
	}
	return false, nil
}
func shouldTryOldModbus(raw []byte) bool {
	// modv is the main prefix, modbus prefix and the respective schema are kept for backwards compatibility
	return bytes.Index(raw, []byte("\"modbus:")) != -1
}
func shouldTryModbus(raw []byte) bool {
	return bytes.Index(raw, []byte("\"modv:")) != -1
}

func ValidateAsTmcImportable(raw []byte, parsed any) (*model.ThingModel, error) {
	err := tmcMandatoryValidator.Validate(parsed)
	if err != nil {
		return nil, err
	}
	tm := &model.ThingModel{}
	err = json.Unmarshal(raw, tm)
	if err != nil {
		return nil, err
	}
	tm.Author.Name = utils.SanitizeName(tm.Author.Name)
	tm.Manufacturer.Name = utils.SanitizeName(tm.Manufacturer.Name)
	tm.Mpn = utils.SanitizeName(tm.Mpn)
	return tm, nil
}

// ValidateThingModel validates the presence of the mandatory fields in the TM to be imported.
// Returns parsed *model.ThingModel, where the author name, manufacturer name, and mpn have been sanitized for use in filenames
func ValidateThingModel(raw []byte) (*model.ThingModel, error) {
	var parsed any
	err := json.Unmarshal(raw, &parsed)
	if err != nil {
		return nil, err
	}

	tm, err := ValidateAsTmcImportable(raw, parsed)
	if err != nil {
		return nil, err
	}

	err = ValidateAsTM(raw, parsed)
	if err != nil {
		return tm, err
	}

	validated, err := ValidateAsModbus(raw, parsed)
	if validated {
		if err != nil {
			return tm, err
		}
	}

	return tm, nil
}
