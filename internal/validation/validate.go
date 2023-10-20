package validation

import (
	_ "embed"
	"github.com/flowstack/go-jsonschema"
	"log/slog"
)

//go:embed tm-json-schema-validation.json
var tmValidationSchema []byte

type TMValidator struct {
	validator *jsonschema.Schema
	log       *slog.Logger
}

func NewTMValidator() (*TMValidator, error) {
	validator, err := jsonschema.New(tmValidationSchema)
	if err != nil {
		return nil, err
	}
	return &TMValidator{validator, slog.Default()}, nil
}
func (v *TMValidator) ValidateTM(tm []byte) error {
	_, err := v.validator.Validate(tm)
	return err
}
