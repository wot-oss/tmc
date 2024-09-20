package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// repoSetConfigCmd represents the 'repo set-config' command
var repoSetConfigCmd = &cobra.Command{
	Use:   "set-config [--type <type>] <name> (<config> | --file <config-file>)",
	Short: "Set config for a repository",
	Long: `Overwrite config of a repository. Depending on the repository type,
the config may be a simple string, like a URL, or a json file.
--type is optional only if --file is used and the type is specified there.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		typ, err := cmd.Flags().GetString("type")
		if err != nil {
			cli.Stderrf("internal error: %v", err)
			os.Exit(1)
		}
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

		descr, _ := cmd.Flags().GetString("description")

		err = cli.RepoSetConfig(name, typ, confStr, confFile, descr)
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
	repoCmd.AddCommand(repoSetConfigCmd)
	repoSetConfigCmd.Flags().StringP("type", "t", "", "type of repo to add")
	_ = repoSetConfigCmd.RegisterFlagCompletionFunc("type", completion.CompleteRepoTypes)
	repoSetConfigCmd.Flags().StringP("file", "f", "", "name of the file to read repo config from")
	repoSetConfigCmd.Flags().StringP("description", "d", "", "description of the repo")
}
