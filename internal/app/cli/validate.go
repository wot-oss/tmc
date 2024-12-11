package cli

import (
	"context"

	"github.com/wot-oss/tmc/internal/commands/validate"
	"github.com/wot-oss/tmc/internal/utils"
)

func ValidateFile(ctx context.Context, filename string) error {
	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		Stderrf("could not read file: %v\n", err)
		return err
	}

	_, err = validate.ValidateThingModel(raw)
	if err != nil {
		Stderrf("validation error: %v\n", err)
		return err
	}
	return nil
}
