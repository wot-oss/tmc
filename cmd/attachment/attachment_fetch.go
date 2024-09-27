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
	Long: `Fetch an attachment to a TM name or a TM ID.

The --concat flag can be used to fetch a concatenation of the attachment to a TM name and 
all homonymous attachments to TMs with this TM name. This can be used, for example, to produce a single README.md or 
CHANGELOG.md file from snippets attached to each TM ID.
Applied to a TM ID --concat flag has no effect.
Concatenating non-text-based attachments is unlikely to produce useful results.  
`,
	Args: cobra.ExactArgs(2),
	Run:  attachmentFetch,
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

	concat, _ := command.Flags().GetBool("concat")
	err := cli.AttachmentFetch(context.Background(), spec, args[0], args[1], concat)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cmd.AddRepoDisambiguatorFlags(attachmentFetchCmd)
	attachmentCmd.AddCommand(attachmentFetchCmd)
	attachmentFetchCmd.Flags().BoolP("concat", "c", false, "Fetch a concatenation of the attachment to a TM name and homonymous attachments to all versions of the same")
}
