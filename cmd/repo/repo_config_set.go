package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoConfigSetCmd represents the 'repo config set' command
var repoConfigSetCmd = &cobra.Command{
	Use:   "set <name> (<config> | --file <config-file>)",
	Short: "Set config for a repository",
	Long: `Set config of a repository. Depending on the repository type, overrides either the location string, 
or the complete config in JSON format as displayed by 'repo show''.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		confStr := ""
		if len(args) > 1 {
			confStr = args[1]
		}

		confFile, err := cmd.Flags().GetString("file")
		if err != nil {
			cli.Stderrf("internal error: %v", err)
			os.Exit(1)
		}

		err = cli.RepoSetConfig(name, confStr, confFile)
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completion.CompleteRepoNames(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	repoConfigCmd.AddCommand(repoConfigSetCmd)
	repoConfigSetCmd.Flags().StringP("file", "f", "", "name of the file to read repo config from")
}
