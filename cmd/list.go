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
E.g. 'MyCompany/BarTech' will not match 'MyCompany/BarTechCorp', but will match 'MyCompany/BarTech/BazLamp'.

Name pattern, filters and search can be combined to narrow down the result.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeList,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(listCmd)
	AddRepoConstraintFlags(listCmd)
	AddTMFilterFlags(listCmd, &listFilterFlags)
}

func executeList(cmd *cobra.Command, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	spec := RepoSpecFromFlags(cmd)

	search := CreateSearchParamsFromCLI(listFilterFlags, name)
	err := cli.List(context.Background(), spec, search)
	if err != nil {
		cli.Stderrf("list failed")
		os.Exit(1)
	}
}
