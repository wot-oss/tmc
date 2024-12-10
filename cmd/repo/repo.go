package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
	Long: `The command repo and its subcommands allow to manage the list of named repositories and their settings.
When no subcommand is given, defaults to list.`,
	Run: repoList,
}
var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List named repositories",
	Long:  `List named repositories`,
	Run:   repoList,
}

func init() {
	cmd.RootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
	cmd.AddOutputFormatFlag(repoCmd)
	cmd.AddOutputFormatFlag(repoListCmd)
}

func repoList(command *cobra.Command, _ []string) {
	format := command.Flag("format").Value.String()
	err := cli.RepoList(format)
	if err != nil {
		os.Exit(1)
	}
}
