package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/repos"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import <file-or-directory>",
	Short: "Import a TM or a directory with TMs into a catalog",
	Long: `Import a single Thing Model or a directory with Thing Models into a catalog.
	
Importing a directory will walk the directory tree recursively and import all found ThingModels.

Specifying the target repository with --directory or --repo is optional if there's exactly one enabled named repository in the config.
`,
	Args: cobra.ExactArgs(1),
	Run:  executeImport,
}

func init() {
	RootCmd.AddCommand(importCmd)
	AddRepoDisambiguatorFlags(importCmd)
	importCmd.Flags().StringP("opt-path", "p", "", "Appends optional path parts to the target path (and id) of imported files, after the mandatory path structure")
	_ = importCmd.RegisterFlagCompletionFunc("repo", completion.NoCompletionNoFile)
	importCmd.Flags().BoolP("opt-tree", "t", false, `Use original directory tree structure below file-or-dirname as --opt-path for each found ThingModel file.
	Has no effect when file-or-dirname points to a file.
	Overrides --opt-path`)
	importCmd.Flags().Bool("force", false, `Force import, even if there are conflicts with existing TMs.`)
}

func executeImport(cmd *cobra.Command, args []string) {
	optPath := cmd.Flag("opt-path").Value.String()
	optTree, _ := cmd.Flags().GetBool("opt-tree")
	force, _ := cmd.Flags().GetBool("force")
	spec := RepoSpecFromFlags(cmd)
	opts := repos.ImportOptions{
		Force:   force,
		OptPath: optPath,
	}
	_, err := cli.NewImportExecutor(time.Now).Import(context.Background(), args[0], spec, optTree, opts)
	if err != nil {
		fmt.Println("import failed")
		os.Exit(1)
	}
}
