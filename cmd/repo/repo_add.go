package repo

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/repos"
)

// repoAddCmd represents the 'repo add' command
var repoAddCmd = &cobra.Command{
	Use:   "add [--type <type>] <name> (<config> | --file <configFileName>)",
	Short: "Add a named repository",
	Long: `Add a named repository to the tmc configuration file. Depending on the repository type,
the config may be a simple string, like a URL, or a json file.
--type is optional only if --file is used and the type is specified there.
`,
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

		err = cli.RepoAdd(name, typ, confStr, confFile, descr)
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
}

func init() {
	repoCmd.AddCommand(repoAddCmd)
	repoAddCmd.Flags().StringP("type", "t", "", fmt.Sprintf("type of repo to add. One of [%s]", strings.Join(repos.SupportedTypes, ", ")))
	_ = repoAddCmd.RegisterFlagCompletionFunc("type", completion.CompleteRepoTypes)
	repoAddCmd.Flags().StringP("file", "f", "", "name of the file to read repo config from")
	repoAddCmd.Flags().StringP("description", "d", "", "description of the repo")
}
