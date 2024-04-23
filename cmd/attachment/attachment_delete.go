package repo

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var attachmentDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an attachment",
	Long:  `Delete an attachment`,
	Args:  cobra.ExactArgs(2),
	Run:   attachmentDelete,
}

func attachmentDelete(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.AttachmentDelete(spec, args[0], args[1])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentDeleteCmd)
}
