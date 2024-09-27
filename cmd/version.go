package cmd

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/config"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show tmc version information",
	Long:  `Show tmc version information`,
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("tmc version %s\n", cli.TmcVersion)
		cf := viper.ConfigFileUsed()
		if cf == "" {
			cf = fmt.Sprintf("No config.json file found in '%s'. Using default settings", config.ConfigDir)
		}
		fmt.Printf("Configuration file used: %s\n", cf)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
