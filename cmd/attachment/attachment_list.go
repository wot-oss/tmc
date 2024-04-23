package repo

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var attachmentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List attachments",
	Long:  `List attachments`,
	Args:  cobra.ExactArgs(1),
	Run:   attachmentList,
}

func attachmentList(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.AttachmentList(spec, args[0])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentListCmd)
}
