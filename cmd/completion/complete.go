package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func CompleteRemoteNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	config, err := remotes.DefaultManager().ReadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var rNames []string
	for k, _ := range config {
		rNames = append(rNames, k)
	}

	return rNames, cobra.ShellCompDirectiveNoFileComp
}

func CompleteRemoteTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return remotes.SupportedTypes, cobra.ShellCompDirectiveNoFileComp
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

	rs, err := getRemote(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	fns, err := rs.ListCompletions(remotes.CompletionKindFetchNames, toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return fns, cobra.ShellCompDirectiveNoFileComp

}

func getRemote(cmd *cobra.Command) (remotes.Remote, error) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()

	spec, err := remotes.NewSpec(remoteName, dirName)
	if err != nil {
		return nil, err
	}

	rs, err := remotes.GetSpecdOrAll(remotes.DefaultManager(), spec)
	if err != nil {
		return nil, err
	}
	return rs, nil
}
func CompleteTMNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	rs, err := getRemote(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	cs, err := rs.ListCompletions(remotes.CompletionKindNames, toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return cs, cobra.ShellCompDirectiveNoFileComp
}
