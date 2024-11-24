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
	Use:   "add <name> [--type <type>] ((<location> [--description <description>]) | --file <config-file> | --json <config-json>)",
	Short: "Add a named repository",
	Long: `Add a named repository to the tmc configuration file. Using <location> is equivalent to passing a json config file with the following content:
{"type": "<type>", "loc": "<location>", "description": "<description>"}. See online user documentation for details on json config file format.
--type is optional only if --file or --json is used and the type is specified in the config file.
--file and --json override --description.
`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		typ := cmd.Flag("type").Value.String()
		name := args[0]

		confFile := cmd.Flag("file").Value.String()
		jsonConf := cmd.Flag("json").Value.String()
		descr := cmd.Flag("description").Value.String()

		locStr := ""
		if len(args) > 1 {
			locStr = args[1]
		}

		err := cli.RepoAdd(name, typ, locStr, descr, jsonConf, confFile)
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
	repoAddCmd.Flags().StringP("file", "f", "", "name of the file containing the repo config")
	repoAddCmd.Flags().StringP("json", "j", "", "repo config in json format")
	repoAddCmd.Flags().StringP("description", "d", "", "description of the repo")
}
