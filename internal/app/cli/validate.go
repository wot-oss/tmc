package cli

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands/validate"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func ValidateFile(filename string) error {

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
	fmt.Printf("validated successfully: %s\n", filename)
	return nil
}
