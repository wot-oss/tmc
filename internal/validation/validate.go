package validation

import (
	_ "embed"
	"encoding/json"
	"github.com/flowstack/go-jsonschema"
	"github.com/go-playground/validator/v10"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"log/slog"
)

//go:embed tm-json-schema-validation.json
var tmValidationSchema []byte

//go:embed modbus.schema.json
var modbusValidationSchema []byte

type TMValidator struct {
	schemaValidator *jsonschema.Schema
	structValidator *validator.Validate
	log             *slog.Logger
}

func NewTMValidator() (*TMValidator, error) {
	schemaValidator, err := jsonschema.New(tmValidationSchema)
	if err != nil {
		return nil, err
	}
	err = schemaValidator.AddSchemaString(string(modbusValidationSchema))
	if err != nil {
		return nil, err
	}
	structValidator := validator.New(validator.WithRequiredStructEnabled())

	return &TMValidator{schemaValidator, structValidator, slog.Default()}, nil
}
func (v *TMValidator) ValidateTM(raw []byte) (*model.ThingModel, error) {
	tm, err := v.validateHasRequiredMetadata(raw)
	if err != nil {
		//fixme: add log or wrap err here
		return nil, err
	}
	v.log.Info("required ThingModel metadata is present")
	err = v.validateAgainstJSONSchema(raw)
	if err != nil {
		//fixme: add log or wrap err here
		return nil, err
	}
	v.log.Info("passed validation against JSON schema for ThingModels")

	return tm, nil
}

func (v *TMValidator) validateHasRequiredMetadata(raw []byte) (*model.ThingModel, error) {
	tm := &model.ThingModel{}
	err := json.Unmarshal(raw, tm)
	if err != nil {
		return nil, err
	}
	err = v.structValidator.Struct(tm)
	if err != nil {
		return nil, err
	}
	return tm, nil
}

func (v *TMValidator) validateAgainstJSONSchema(raw []byte) error {
	_, err := v.schemaValidator.Validate(raw)
	return err
}
