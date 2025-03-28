package repo

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var attachmentDeleteCmd = &cobra.Command{
	Use:   "delete <tm-name-or-id> <attachment-name>",
	Short: "Delete an attachment",
	Long:  `Delete an attachment`,
	Args:  cobra.ExactArgs(2),
	Run:   attachmentDelete,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		switch len(args) {
		case 0:
			return completion.CompleteTMNamesOrIds(cmd, args, toComplete)
		case 1:
			return completion.CompleteAttachmentNames(cmd, args, toComplete)
		default:
			comps = cobra.AppendActiveHelp(comps, "This command does not take any more arguments")
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func attachmentDelete(command *cobra.Command, args []string) {
	spec := cmd.RepoSpecFromFlags(command)

	err := cli.AttachmentDelete(context.Background(), spec, args[0], args[1])
	if err != nil {
		cli.Stderrf("attachment delete failed")
		os.Exit(1)
	}
}

func init() {
	cmd.AddRepoDisambiguatorFlags(attachmentDeleteCmd)
	attachmentCmd.AddCommand(attachmentDeleteCmd)
}
