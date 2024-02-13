package cmd

import (
	"fmt"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the tm-catalog-cli version information",
	Long:  `Show the tm-catalog-cli version information.`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tm-catalog-cli version %s\n", cli.TmcVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
