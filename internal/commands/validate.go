package commands

import (
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/model"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/validation"
	"log/slog"
	"os"
)

func ValidateThingModel(raw []byte) (*model.ThingModel, error) {
	var log = slog.Default()

	tmValidator, err := validation.NewTMValidator()
	if err != nil {
		log.Error("could not create TM validator", "error", err)
		return nil, fmt.Errorf("could not create TM validator: %w", err)
	}
	tm, err := tmValidator.ValidateTM(raw)
	if err != nil {
		log.Error("validation failed", "error", err)
		os.Exit(1)
	}
	return tm, nil
}
