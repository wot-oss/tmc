package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var listCmd = &cobra.Command{
	Use:   "list [PATTERN]",
	Short: "List TMs in catalog",
	Long:  `List TMs and filter for PATTERN in all mandatory fields`,
	Args:  cobra.MaximumNArgs(1),
	Run:   executeList,
}

func init() {
	RootCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("remote", "r", "", "name of the remote to list")
	listCmd.Flags().StringP("directory", "d", "", "TM repository directory to list")
}

func executeList(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}
	spec, err := remotes.NewSpec(remoteName, dirName)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.List(spec, filter)
	if err != nil {
		os.Exit(1)
	}
}
