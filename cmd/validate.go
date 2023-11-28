package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate a TM before importing",
	Long:  `validate a ThingModel to ensure it is ready to be imported into TM catalog`,
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.ValidateFile(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(validateCmd)
}
