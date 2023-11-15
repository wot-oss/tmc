package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var listCmd = &cobra.Command{
	Use:   "list [PATTERN]",
	Short: "List TMs in catalog",
	Long:  `List TMs and filter for PATTERN in all mandatory fields`,
	Args:  cobra.MaximumNArgs(1),
	Run:   listRemote,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func listRemote(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()

	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	err := cli.ListRemote(remoteName, filter)
	if err != nil {
		os.Exit(1)
	}
}
