package cli

import (
	"fmt"

	"github.com/wot-oss/tmc/internal/commands"
	"github.com/wot-oss/tmc/internal/utils"
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
