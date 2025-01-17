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

var copyFilterFlags = FilterFlags{}

var copyCmd = &cobra.Command{
	Use:   "copy [<name-pattern>]",
	Short: "Copy multiple TMs and their attachments from one repository to another",
	Long: `Copy TMs from one repository to another, selecting by name pattern, filters or search. 

Accepts the same <name-pattern> and filter flags as list command.
Use list command with the same parameters to verify beforehand which TMs are going to be copied.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeCopy,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(copyCmd)
	AddRepoConstraintFlags(copyCmd)
	AddOutputFormatFlag(copyCmd)
	copyCmd.Flags().StringP("toRepo", "R", "", "Name of the target repository. Mutually exclusive with --toDirectory. Required, unless --toDirectory is set.")
	_ = copyCmd.RegisterFlagCompletionFunc("toRepo", completion.CompleteRepoNames)
	copyCmd.Flags().StringP("toDirectory", "D", "", "Use the specified directory as the target repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository. Mutually exclusive with --toRepo. Required, unless --toRepo is set.")
	_ = copyCmd.MarkFlagDirname("toDirectory")
	AddTMFilterFlags(copyCmd, &copyFilterFlags)
	copyCmd.Flags().Bool("force", false, `Force copy, even if there are conflicts with existing TMs.`)
	copyCmd.Flags().Bool("ignore-existing", false, `Ignore TMs and attachments that have conflicts with existing ones instead of returning an error code.`)
}

func executeCopy(cmd *cobra.Command, args []string) {
	toRepoName := cmd.Flag("toRepo").Value.String()
	toDirName := cmd.Flag("toDirectory").Value.String()
	force, _ := cmd.Flags().GetBool("force")
	ie, _ := cmd.Flags().GetBool("ignore-existing")
	format := cmd.Flag("format").Value.String()

	spec := RepoSpecFromFlags(cmd)
	toSpec, err := model.NewSpec(toRepoName, toDirName)
	if errors.Is(err, model.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --toRepo and --toDirectory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	search := CreateFiltersFromCLI(copyFilterFlags, name)
	err = cli.Copy(context.Background(), spec, toSpec, search, repos.ImportOptions{Force: force, IgnoreExisting: ie}, format)

	if err != nil {
		cli.Stderrf("copy failed")
		os.Exit(1)
	}
}
