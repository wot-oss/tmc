package cmd

import (
	"errors"
	"log/slog"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/cmd/completion"
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

func preRunAll(cmd *cobra.Command, _ []string) {
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

func RepoSpecFromFlags(cmd *cobra.Command) model.RepoSpec {
	repoName := cmd.Flag("repo").Value.String()
	dir := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dir)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}
	return spec
}

type FilterFlags struct {
	FilterAuthor       string
	FilterManufacturer string
	FilterMpn          string
	Search             string
	Deep               bool
}

func CreateSearchParamsFromCLI(flags FilterFlags, name string) *model.SearchParams {
	return model.ToSearchParams(&flags.FilterAuthor, &flags.FilterManufacturer, &flags.FilterMpn, &name, &flags.Search,
		&model.SearchOptions{NameFilterType: model.PrefixMatch, UseBleve: flags.Deep})
}

// AddRepoConstraintFlags adds repo and directory flags for commands that can use multiple repositories (e.g. list)
func AddRepoConstraintFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("repo", "r", "", "Name of the repository to use as source. Uses all if omitted. Mutually exclusive with --directory.")
	_ = cmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	cmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository. Mutually exclusive with --repo.")
	_ = cmd.MarkFlagDirname("directory")
}

// AddRepoDisambiguatorFlags adds repo and directory flags for commands that use exactly one repository (e.g. delete)
func AddRepoDisambiguatorFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("repo", "r", "", "Name of the repository to use. Required if the repository is ambiguous. Mutually exclusive with --directory.")
	_ = cmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	cmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository. Mutually exclusive with --repo.")
	_ = cmd.MarkFlagDirname("directory")
}

func AddTMFilterFlags(cmd *cobra.Command, flags *FilterFlags) {
	cmd.Flags().StringVar(&flags.FilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	cmd.Flags().StringVar(&flags.FilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	cmd.Flags().StringVar(&flags.FilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	cmd.Flags().StringVarP(&flags.Search, "search", "s", "", "search TMs by their content matching the search term")
	cmd.Flags().BoolVar(&flags.Deep, "deep", false, "use bleve query index in search flag for more precise matching")
}
