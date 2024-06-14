package repo

import (
	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd"
	"github.com/wot-oss/tmc/cmd/completion"
)

var attachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Manage TM attachments",
	Long: `The subcommands of the attachment command allow to manage the attachments to TMs.
You can attach files to a single TM version, or to an inventory name, encompassing all TM versions for a device.
For all attachment operations you must unambiguously specify the repository.`,
}

func init() {
	cmd.RootCmd.AddCommand(attachmentCmd)
	attachmentCmd.PersistentFlags().StringP("repo", "r", "", "Use the named repository for the operation.")
	_ = attachmentCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	attachmentCmd.PersistentFlags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = attachmentCmd.MarkFlagDirname("directory")
}
