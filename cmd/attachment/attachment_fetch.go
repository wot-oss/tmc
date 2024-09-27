package repo

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var attachmentFetchCmd = &cobra.Command{
	Use:   "fetch <tm-name-or-id> <attachment-name>",
	Short: "Fetch an attachment",
	Long:  `Fetch an attachment`,
	Args:  cobra.ExactArgs(2),
	Run:   attachmentFetch,
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

func attachmentFetch(command *cobra.Command, args []string) {
	spec := cmd.RepoSpecFromFlags(command)

	err := cli.AttachmentFetch(context.Background(), spec, args[0], args[1])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cmd.AddRepoDisambiguatorFlags(attachmentFetchCmd)
	attachmentCmd.AddCommand(attachmentFetchCmd)
}
