package validation

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qri-io/jsonschema"
	"log/slog"
)

//go:embed td-json-schema-validation.json
var tdValidationSchema []byte

type TMValidator struct {
	jsSchemaValidator *jsonschema.Schema
	log               *slog.Logger
}

func NewTMValidator() (*TMValidator, error) {
	sd := &jsonschema.Schema{}
	err := json.Unmarshal(tdValidationSchema, sd)
	if err != nil {
		return nil, err
	}
	return &TMValidator{sd, slog.Default()}, nil
}
func (v *TMValidator) ValidateTM(tm []byte) error {
	var errs []jsonschema.KeyError
	var err error
	//errs, err = v.jsSchemaValidator.ValidateBytes(context.TODO(), tm) // fixme: enable the actual validation
	if err != nil {
		return fmt.Errorf("could not validate TM against JSON schema: %w", err)
	}
	if len(errs) == 0 {
		v.log.Info("passed validation against JSON schema for ThingDescriptions")
		return nil
	} else {
		stdErrs := []error{}
		for _, e := range errs {
			stdErrs = append(stdErrs, e)
			v.log.Error("validation error", "error", e)
		}

		return errors.Join(stdErrs...)
	}

}
