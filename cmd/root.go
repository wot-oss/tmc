package cmd

import (
	"fmt"
	"log/slog"
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

var loglevel string
var logEnabledDefaultCmd = []string{"serve"}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	cf := viper.ConfigFileUsed()
	if cf == "" {
		cf = "No config.json file found in ~/.tm-catalog or workdir. Using default settings"
	}
	RootCmd.Long = RootCmd.Long + fmt.Sprintf("\n\nConfiguration file used: %s", cf)
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
	RootCmd.PersistentFlags().StringVarP(&loglevel, "loglevel", "l", "", "enable logging by setting a log level, one of [error, warn, info, debug, off]")
	RootCmd.PersistentPreRun = preRunAll
	config.InitViper()
	// bind viper variable "loglevel" to CLI flag --loglevel of root command
	_ = viper.BindPFlag(config.KeyLogLevel, RootCmd.PersistentFlags().Lookup("loglevel"))
}

func preRunAll(cmd *cobra.Command, args []string) {
	// set default loglevel depending on subcommand
	logDefault := cmd != nil && slices.Contains(logEnabledDefaultCmd, cmd.CalledAs())
	if logDefault {
		viper.SetDefault(config.KeyLogLevel, slog.LevelInfo.String())
	} else {
		viper.SetDefault(config.KeyLogLevel, config.LogLevelOff)
	}

	internal.InitLogging()
}
