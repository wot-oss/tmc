package cmd

import (
	"errors"
	"log/slog"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "tmc",
	Short: "A CLI client for TM catalogs",
	Long: `tmc is a CLI client for contributing to and searching
ThingModel catalogs.`,
}

var cfgFile string
var loglevel string
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

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", config.DefaultConfigDir, "path to config directory")
	RootCmd.PersistentFlags().StringVarP(&loglevel, "loglevel", "l", "", "enable logging by setting a log level, one of [error, warn, info, debug, off]")
	RootCmd.PersistentPreRun = preRunAll
	// bind viper variable "loglevel" to CLI flag --loglevel of root command
	_ = viper.BindPFlag(config.KeyLogLevel, RootCmd.PersistentFlags().Lookup("loglevel"))
	_ = viper.BindPFlag(config.KeyConfigPath, RootCmd.PersistentFlags().Lookup("config"))
}

func preRunAll(cmd *cobra.Command, args []string) {
	config.ReadInConfig()
	// set default loglevel depending on subcommand
	logDefault := cmd != nil && slices.Contains(logEnabledDefaultCmd, cmd.CalledAs())
	if logDefault {
		viper.SetDefault(config.KeyLogLevel, slog.LevelInfo.String())
	} else {
		viper.SetDefault(config.KeyLogLevel, config.LogLevelOff)
	}
	internal.InitLogging()
}

func RepoSpec(cmd *cobra.Command) model.RepoSpec {
	repoName := cmd.Flag("repo").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dir)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	return spec
}
