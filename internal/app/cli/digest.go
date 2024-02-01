package cli

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"
)

func CalcFileDigest(filename string) error {

	_, raw, err := utils.ReadRequiredFile(filename)
	if err != nil {
		fmt.Printf("could not read file: %v\n", err)
		return err
	}

	s, _, err := commands.CalculateFileDigest(raw)
	if err != nil {
		Stderrf("error: %v\n", err)
		return err
	}
	fmt.Println(s)
	return nil
}
