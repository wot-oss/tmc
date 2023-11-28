package remote

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteAddCmd represents the 'remote add' command
var remoteAddCmd = &cobra.Command{
	Use:   "add [--type <type>] <name> (<config> | --file <configFileName>)",
	Short: "Add a remote repository",
	Long: `Add a remote repository to the tm-catalog-cli configuration file. Depending on the remote type,
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

		err = cli.RemoteAdd(name, typ, confStr, confFile)
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
	remoteAddCmd.Flags().StringP("type", "t", "", "type of remote to add")
	remoteAddCmd.Flags().StringP("file", "f", "", "name of the file to read remote config from")
}
