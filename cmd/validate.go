package cmd

import (
	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
	"log/slog"
	"os"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "validate a TM before importing",
	Long:  `validate a ThingModel to ensure it is ready to be imported into TM catalog`,
	Run: func(cmd *cobra.Command, args []string) {
		var log = slog.Default()

		log.Debug("executing validate", "args", args)
		_, raw := commands.ReadRequiredFile(args[0])

		_, err := commands.ValidateThingModel(raw)
		if err != nil {
			log.Error("validation failed", "error", err)
			os.Exit(1)
		}
		log.Info("TM is valid")
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
