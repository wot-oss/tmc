package validate

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/kennygrant/sanitize"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

//go:embed tm-json-schema-validation.json
var tmValidationSchema string

//go:embed modbus.schema.json
var modbusValidationSchema string

//go:embed tmc-mandatory.schema.json
var tmcMandatorySchema string

var tmcMandatoryValidator *jsonschema.Schema
var tmValidator *jsonschema.Schema
var modbusValidator *jsonschema.Schema

const (
	tmcMandatorySchemaUrl = "resource://tmc-mandatory.schema.json"
	tmSchemaUrl           = "https://raw.githubusercontent.com/w3c/wot-thing-description/main/validation/tm-json-schema-validation.json"
	modbusSchemaUrl       = "resource://modbus.schema.json"
)

func init() {
	tmcMandatoryValidator = jsonschema.MustCompileString(tmcMandatorySchemaUrl, tmcMandatorySchema)
	tmValidator = jsonschema.MustCompileString(tmSchemaUrl, tmValidationSchema)

	modbusCompiler := jsonschema.NewCompiler()
	err := modbusCompiler.AddResource(tmSchemaUrl, strings.NewReader(tmValidationSchema))
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
	return false, nil
}
func shouldTryModbus(raw []byte) bool {
	return bytes.Index(raw, []byte("\"modbus:")) != -1
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
	tm.Author.Name = sanitize.BaseName(tm.Author.Name)
	tm.Manufacturer.Name = sanitize.BaseName(tm.Manufacturer.Name)
	tm.Mpn = sanitize.BaseName(tm.Mpn)
	return tm, nil
}

// ValidateThingModel validates the presence of the mandatory fields in the TM to be imported.
// Returns parsed *model.ThingModel, where the author name, manufacturer name, and mpn have been sanitized for use in filenames
func ValidateThingModel(raw []byte) (*model.ThingModel, error) {
	log := slog.Default()

	var parsed any
	err := json.Unmarshal(raw, &parsed)
	if err != nil {
		return nil, err
	}

	tm, err := ValidateAsTmcImportable(raw, parsed)
	if err != nil {
		return nil, err
	}
	log.Info("required Thing Model metadata is present")

	err = ValidateAsTM(raw, parsed)
	if err != nil {
		return tm, err
	}
	log.Info("passed validation against JSON schema for Thing Models")

	validated, err := ValidateAsModbus(raw, parsed)
	if validated {
		if err != nil {
			return tm, err
		} else {
			log.Info("passed validation against JSON schema for Modbus protocol binding")
		}
	}

	return tm, nil
}
