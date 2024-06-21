package repo

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var attachmentListCmd = &cobra.Command{
	Use:   "list <tmNameOrId>",
	Short: "List attachments",
	Long:  `List attachments to given inventory TM name or id`,
	Args:  cobra.ExactArgs(1),
	Run:   attachmentList,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		switch len(args) {
		case 0:
			return completion.CompleteTMNamesOrIds(cmd, args, toComplete)
		default:
			comps = cobra.AppendActiveHelp(comps, "This command does not take any more arguments")
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func attachmentList(command *cobra.Command, args []string) {
	spec := cmd.RepoSpec(command)

	err := cli.AttachmentList(context.Background(), spec, args[0])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentListCmd)
}
