package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <TMID>",
	Short: "Delete a TM by id",
	Long: `Delete a TM by id. Normally, the catalog is write-only and no TMs are ever deleted.
The delete function is implemented for the rare cases when a TM has been pushed whilst containing major errors 
or by mistake. Therefore, it is mandatory to provide the flag --force=true to delete a TM.`,
	Args:              cobra.ExactArgs(1),
	Run:               executeDelete,
	ValidArgsFunction: nil, // just to make explicit in code that a completion function is not wanted here. Don't want to do anything to make deletion easier
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("remote", "r", "", "name of the remote to delete from")
	_ = deleteCmd.RegisterFlagCompletionFunc("remote", completion.CompleteRemoteNames)
	deleteCmd.Flags().StringP("directory", "d", "", "TM repository directory")
	_ = deleteCmd.MarkFlagDirname("directory")
	deleteCmd.Flags().String("force", "", "force the deletion") // intentionally a string flag, not boolean, so that the user has to make that much extra effort to type
}

func executeDelete(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	dirName := cmd.Flag("directory").Value.String()
	force := cmd.Flag("force").Value.String()

	spec, err := remotes.NewSpec(remoteName, dirName)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of target repository. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	if force != "true" {
		cli.Stderrf("Cannot delete a TM unless --force is set to \"true\"")
		os.Exit(1)
	}

	err = cli.Delete(spec, args[0])
	if err != nil {
		cli.Stderrf("delete failed")
		os.Exit(1)
	}
}
