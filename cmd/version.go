package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var TmcVersion = "n/a"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the tm-catalog-cli version information",
	Long:  `Show the tm-catalog-cli version information.`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", TmcVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
