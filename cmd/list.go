package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var listFilterFlags = FilterFlags{}

var listCmd = &cobra.Command{
	Use:   "list [<name-pattern>]",
	Short: "List TMs in catalog",
	Long: `List TMs in catalog by name pattern, filters or search. 
The <name-pattern> can be a full name or a prefix consisting of complete path parts. 
E.g. 'my-company/bar-tech' will not match 'my-company/bar-tech-corp', but will match 'my-company/bar-tech/baz-lamp'.

<name-pattern> and filters can be combined to narrow down the result.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeList,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(listCmd)
	AddRepoConstraintFlags(listCmd)
	AddTMFilterFlags(listCmd, &listFilterFlags)
	AddOutputFormatFlag(listCmd)
}

func executeList(cmd *cobra.Command, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	spec := RepoSpecFromFlags(cmd)
	format := cmd.Flag("format").Value.String()

	search := CreateFiltersFromCLI(listFilterFlags, name)
	err := cli.List(context.Background(), spec, search, format)
	if err != nil {
		cli.Stderrf("list failed")
		os.Exit(1)
	}
}
