package commands

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/validation"
	"log/slog"
)

func ValidateThingModel(raw []byte) (*model.ThingModel, error) {
	log := slog.Default()

	tm, err := validation.ParseRequiredMetadata(raw)
	if err != nil {
		return nil, err
	}
	log.Info("required Thing Model metadata is present")

	err = validation.ValidateAsTM(raw)
	if err != nil {
		return tm, err
	}
	log.Info("passed validation against JSON schema for Thing Models")

	validated, err := validation.ValidateAsModbus(raw)
	if validated {
		if err != nil {
			log.Info("failed [optional] validation against JSON schema for Modbus protocol binding", "error", err)
		} else {
			log.Info("passed validation against JSON schema for Modbus protocol binding")
		}
	}

	return tm, nil
}
