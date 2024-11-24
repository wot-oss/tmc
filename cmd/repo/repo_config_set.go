package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoConfigSetCmd represents the 'repo config set' command
var repoConfigSetCmd = &cobra.Command{
	Use:   "set <name> (<location> | --file <config-file> | --json <config-json>)",
	Short: "Set config for a repository",
	Long: `Set config of a repository. Overrides either just the location string, or the entire config in JSON format as displayed by 'repo show'', except type.
The repository's type cannot be changed.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		locStr := ""
		if len(args) > 1 {
			locStr = args[1]
		}

		confFile := cmd.Flag("file").Value.String()
		jsonConf := cmd.Flag("json").Value.String()

		err := cli.RepoSetConfig(name, locStr, jsonConf, confFile)
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
	repoConfigSetCmd.Flags().StringP("file", "f", "", "name of the file containing repo config")
	repoConfigSetCmd.Flags().StringP("json", "j", "", "repo config in json format")
}
