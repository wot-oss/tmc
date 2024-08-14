package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

var copyFilterFlags = cli.FilterFlags{}

var copyCmd = &cobra.Command{
	Use:   "copy <NAME PATTERN>",
	Short: "Copy multiple TMs and their attachments from one repository to another",
	Long: `Copies TMs from one repository to another, selecting by name pattern, filters or search. 
The name can be a full name or a prefix consisting of complete path parts. 
E.g. 'MyCompany/BarTech' will not match 'MyCompany/BarTechCorp', but will match 'MyCompany/BarTech/BazLamp'.

Name pattern, filters and search can be combined to narrow down the result.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeCopy,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(copyCmd)
	copyCmd.Flags().StringP("repo", "r", "", "Name of the source repository. Copies from all repositories if omitted")
	_ = copyCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	copyCmd.Flags().StringP("directory", "d", "", "Use the specified directory as source repository. The directory must contain a tmc repository.")
	_ = copyCmd.MarkFlagDirname("directory")
	copyCmd.Flags().StringP("toRepo", "R", "", "Name of the target repository for copying")
	_ = copyCmd.RegisterFlagCompletionFunc("toRepo", completion.CompleteRepoNames)
	copyCmd.Flags().StringP("toDirectory", "D", "", "Use the specified directory as target repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = copyCmd.MarkFlagDirname("toDirectory")
	copyCmd.Flags().StringVar(&copyFilterFlags.FilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	copyCmd.Flags().StringVar(&copyFilterFlags.FilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	copyCmd.Flags().StringVar(&copyFilterFlags.FilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	copyCmd.Flags().StringVarP(&copyFilterFlags.Search, "search", "s", "", "search TMs by their content matching the search term")
	copyCmd.Flags().Bool("force", false, `Force copy, even if there are conflicts with existing TMs.`)
}

func executeCopy(cmd *cobra.Command, args []string) {
	toRepoName := cmd.Flag("toRepo").Value.String()
	toDirName := cmd.Flag("toDirectory").Value.String()
	force, _ := cmd.Flags().GetBool("force")

	spec := RepoSpec(cmd)
	toSpec, err := model.NewSpec(toRepoName, toDirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --toRepo and --toDirectory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	search := cli.CreateSearchParamsFromCLI(copyFilterFlags, name)
	err = cli.Copy(context.Background(), spec, toSpec, search, repos.ImportOptions{Force: force})

	if err != nil {
		cli.Stderrf("copy failed")
		os.Exit(1)
	}
}
