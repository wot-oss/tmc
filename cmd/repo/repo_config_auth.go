package repo

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/repos"
)

// repoConfigAuthCmd represents the 'repo config auth' command
var repoConfigAuthCmd = &cobra.Command{
	Use:   "auth <repo-name> <auth-type> [<auth-string>|(<auth-key>=<auth-value> ...)]",
	Short: "Set authentication config for a repository",
	Long: `Set auth config of a repository. <auth-type> must be one of: none, bearer, basic.
Authentication data is set as a single string or a set of key-value pairs. The possible keys depend on the <auth-type>
and are listed in the following table:

|   auth-type              |                    keys                     |
|--------------------------|---------------------------------------------|
| none                     |                                             |
| bearer                   | token                                       |
| basic                    | username, password                          |

For auth type 'bearer', because it has a single possible key, prefixing the token value with 'token=' can be omitted.
`,
	Example: "tmc repo config auth http-repo bearer qfdhjf83cblkju",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoName := args[0]
		authType := args[1]
		var authData []string
		if len(args) > 2 {
			authData = args[2:]
		}
		err := cli.RepoSetAuth(repoName, authType, authData)
		if err != nil {
			_ = cmd.Usage()
			os.Exit(1)
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			return completion.CompleteRepoNames(cmd, args, toComplete)
		case 1:
			return []string{repos.AuthMethodNone, repos.AuthMethodBasic, repos.AuthMethodBearerToken /*, repos.AuthMethodOauthClientCredentials*/}, cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	repoConfigCmd.AddCommand(repoConfigAuthCmd)
}
