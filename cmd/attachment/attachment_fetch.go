package repo

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var attachmentFetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch an attachment",
	Long:  `Fetch an attachment`,
	Args:  cobra.ExactArgs(2),
	Run:   attachmentFetch,
}

func attachmentFetch(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.AttachmentFetch(spec, args[0], args[1])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentFetchCmd)
}
