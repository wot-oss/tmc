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
	cmd.RootCmd.AddCommand(repoListCmd)
}

func repoList(cmd *cobra.Command, args []string) {
	err := cli.RepoList()
	if err != nil {
		os.Exit(1)
	}
}
