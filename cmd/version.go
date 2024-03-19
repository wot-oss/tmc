package cmd

import (
	"fmt"

	"github.com/wot-oss/tmc/internal/app/cli"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the tmc version information",
	Long:  `Show the tmc version information.`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tmc version %s\n", cli.TmcVersion)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
