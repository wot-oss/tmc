package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

// remoteAddCmd represents the 'remote add' command
var remoteAddCmd = &cobra.Command{
	Use:   "add --name <name> --type <type> (<config> | --file <configFileName>)",
	Short: "Add a remote repository",
	Long: `Add a remote repository to the tm-catalog configuration file. Depending on the remote type,
the config may be a simple string, like a URL string, or a json file.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			cli.Stderrf("internal error: %v", err)
			os.Exit(1)
		}
		typ, err := cmd.Flags().GetString("type")
		if err != nil {
			cli.Stderrf("internal error: %v", err)
			os.Exit(1)
		}

		confStr := ""
		if len(args) > 0 {
			confStr = args[0]
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
	remoteAddCmd.Flags().StringP("name", "n", "", "name of remote to add")
	remoteAddCmd.Flags().StringP("type", "t", "", "type of remote to add")
	remoteAddCmd.Flags().StringP("file", "f", "", "name of the file to read remote config from")
}
