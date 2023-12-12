package cmd

import (
	"os"
	"slices"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "tm-catalog-cli",
	Short: "A CLI client for TM catalogs",
	Long: `tm-catalog-cli is a CLI client for contributing to and searching
ThingModel catalogs.`,
}

var log bool
var logEnabledDefaultCmd = []string{"serve"}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tm-catalog-cli.yaml)")
	RootCmd.PersistentFlags().BoolVarP(&log, "log", "l", false, "enable logging")
	RootCmd.PersistentPreRun = preRunAll
	// bind viper variable "log" to CLI flag --log of root command
	_ = viper.BindPFlag(config.KeyLog, RootCmd.PersistentFlags().Lookup("log"))
}

func preRunAll(cmd *cobra.Command, args []string) {
	// set default logging enabled/disabled depending on subcommand
	logDefault := cmd != nil && slices.Contains(logEnabledDefaultCmd, cmd.CalledAs())
	viper.SetDefault(config.KeyLog, logDefault)

	internal.InitLogging()
}
