package repo

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var attachmentPutCmd = &cobra.Command{
	Use:   "put",
	Short: "Put an attachment",
	Long:  `Add or replace an attachment`,
	Args:  cobra.ExactArgs(2),
	Run:   attachmentPut,
}

func attachmentPut(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.AttachmentPut(spec, args[0], args[1])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentPutCmd)
}
