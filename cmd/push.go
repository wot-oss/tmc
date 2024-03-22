package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push <file-or-dirname>",
	Short: "Push a TM or a directory with TMs to catalog",
	Long: `Push a single Thing Model or a directory with Thing Models to a catalog.
file-or-dirname
	The name of the file or directory to push. Pushing a directory will walk the directory tree recursively and 
	push all found ThingModels.

Specifying the target repository with --directory or --repo is optional if there's exactly one enabled named catalog in the config
`,
	Args: cobra.ExactArgs(1),
	Run:  executePush,
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringP("repo", "r", "", "Name of the target repository. Can be omitted if there's only one")
	_ = pushCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	pushCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = pushCmd.MarkFlagDirname("directory")
	pushCmd.Flags().StringP("opt-path", "p", "", "Appends optional path parts to the target path (and id) of imported files, after the mandatory path structure")
	_ = pushCmd.RegisterFlagCompletionFunc("repo", completion.NoCompletionNoFile)
	pushCmd.Flags().BoolP("opt-tree", "t", false, `Use original directory tree structure below file-or-dirname as --opt-path for each found ThingModel file.
	Has no effect when file-or-dirname points to a file.
	Overrides --opt-path`)
}

func executePush(cmd *cobra.Command, args []string) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	optPath := cmd.Flag("opt-path").Value.String()
	optTree, _ := cmd.Flags().GetBool("opt-tree")
	spec, err := model.NewSpec(repoName, dirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --repo and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	results, err := cli.NewPushExecutor(time.Now).Push(context.Background(), args[0], spec, optPath, optTree)
	for _, res := range results {
		fmt.Println(res)
	}
	if err != nil {
		fmt.Println("push failed")
		os.Exit(1)
	}
}
