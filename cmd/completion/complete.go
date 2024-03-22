package completion

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
)

func CompleteRepoNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	config, err := repos.ReadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var rNames []string
	for k, _ := range config {
		rNames = append(rNames, k)
	}

	return rNames, cobra.ShellCompDirectiveNoFileComp
}

func CompleteRepoTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return repos.SupportedTypes, cobra.ShellCompDirectiveNoFileComp
}

func NoCompletionNoFile(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func CompleteFetchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if !strings.Contains(toComplete, ":") {
		names, dir := CompleteTMNames(cmd, args, toComplete)
		if dir != cobra.ShellCompDirectiveError {
			return names, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
		}
		return names, dir
	}

	rs, err := getRepo(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	fns := rs.ListCompletions(context.Background(), repos.CompletionKindFetchNames, toComplete)
	return fns, cobra.ShellCompDirectiveNoFileComp

}

func getRepo(cmd *cobra.Command) (*repos.Union, error) {
	repoName := cmd.Flag("repo").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	spec, err := model.NewSpec(repoName, dirName)
	if err != nil {
		return nil, err
	}

	u, err := repos.GetSpecdOrAll(spec)
	if err != nil {
		return nil, err
	}
	return u, nil
}
func CompleteTMNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	rs, err := getRepo(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cs := rs.ListCompletions(context.Background(), repos.CompletionKindNames, toComplete)
	return cs, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
