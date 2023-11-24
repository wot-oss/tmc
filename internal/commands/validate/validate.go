package validate

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log/slog"

	"github.com/kennygrant/sanitize"
	"github.com/web-of-things-open-source/go-jsonschema"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
)

//go:embed tm-json-schema-validation.json
var tmValidationSchema []byte

//go:embed modbus.schema.json
var modbusValidationSchema []byte

//go:embed tmc-mandatory.schema.json
var tmcMandatorySchema []byte

var tmcMandatoryValidator *jsonschema.Schema
var tmValidator *jsonschema.Schema
var modbusValidator *jsonschema.Schema

func init() {
	var err error
	tmcMandatoryValidator, err = jsonschema.New(tmcMandatorySchema)
	if err != nil {
		panic(err)
	}

	tmValidator, err = jsonschema.New(tmValidationSchema)
	if err != nil {
		panic(err)
	}
	modbusValidator, err = jsonschema.New(modbusValidationSchema)
	if err != nil {
		panic(err)
	}
	err = modbusValidator.AddSchemaString(string(tmValidationSchema))
	if err != nil {
		panic(err)
	}

}

func ValidateAsTM(raw []byte) error {
	_, err := tmValidator.Validate(raw)
	return err
}

// ValidateAsModbus validates a file against modbus protocol binding json schema, but only if it determines that
// the file purports to describe a modbus device.
// Returns a flag indicating whether validate has been attempted and an error if it was not successful
func ValidateAsModbus(raw []byte) (bool, error) {
	if shouldTryModbus(raw) {
		_, err := modbusValidator.Validate(raw)
		return true, err
	}
	return false, nil
}
func shouldTryModbus(raw []byte) bool {
	return bytes.Index(raw, []byte("\"modbus:")) != -1
}

func ValidateAsTmcImportable(raw []byte) (*model.ThingModel, error) {
	_, err := tmcMandatoryValidator.Validate(raw)
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

	tm, err := ValidateAsTmcImportable(raw)
	if err != nil {
		return nil, err
	}
	log.Info("required Thing Model metadata is present")

	err = ValidateAsTM(raw)
	if err != nil {
		return tm, err
	}
	log.Info("passed validation against JSON schema for Thing Models")

	validated, err := ValidateAsModbus(raw)
	if validated {
		if err != nil {
			return tm, err
		} else {
			log.Info("passed validation against JSON schema for Modbus protocol binding")
		}
	}

	return tm, nil
}
