package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/commands"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push file-or-dirname [--remote=remote-name] [--with-path=optional/path] [--copy-tree]",
	Short: "Push a TM or directory with TMs to remote",
	Long: `Push a single ThingModel or an directory with ThingModels to remote catalog.
file-or-dirname
	The name of the file or directory to push. Pushing a directory will walk the directory tree recursively and 
	import all found ThingModels.

--remote, -r
	Name of the target remote repository

--opt-path, -p
	Appends optional path parts to the target path (and id) of imported files, after the mandatory path structure.

--opt-tree, -t
	Use original directory tree structure below file-or-dirname as --opt-path for each found ThingModel file.
	Has no effect when file-or-dirname points to a file.
	Overrides --opt-path.
`,
	Args: cobra.ExactArgs(1),
	Run:  executePush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
	pushCmd.Flags().StringP("opt-path", "p", "", "append optional path to mandatory target directory structure")
	pushCmd.Flags().BoolP("opt-tree", "t", false, "use original directory tree as optional path for each file. Has no effect with a single file. Overrides -p")
}

func executePush(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	log.Debug("executing push", "args", args)
	remoteName := cmd.Flag("remote").Value.String()
	optPath := cmd.Flag("opt-path").Value.String()
	optTree, _ := cmd.Flags().GetBool("opt-tree")
	err := commands.PushToRemote(args[0], remoteName, optPath, optTree)
	if err != nil {
		log.Error("push failed", "error", err)
		os.Exit(1)
	}
}
