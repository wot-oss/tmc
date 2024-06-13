package repo

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

var attachmentFetchCmd = &cobra.Command{
	Use:   "fetch <tmNameOrId> <attachmentName>",
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

func attachmentFetch(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	err = cli.AttachmentFetch(context.Background(), spec, args[0], args[1])
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	attachmentCmd.AddCommand(attachmentFetchCmd)
}