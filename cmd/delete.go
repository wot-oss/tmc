package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
	"github.com/wot-oss/tmc/internal/commands"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <tmid>",
	Short: "Delete a TM by id",
	Long: `Delete a TM by id. Normally, the catalog is write-only and no TMs are ever deleted.
The delete function is implemented for the rare cases when a TM has been imported whilst containing major errors 
or by mistake. Therefore, it is mandatory to provide the flag --force=true to delete a TM.`,
	Args:              cobra.ExactArgs(1),
	Run:               executeDelete,
	ValidArgsFunction: nil, // just to make explicit in code that a completion function is not wanted here. Don't want to do anything to make deletion easier
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	AddRepoDisambiguatorFlags(deleteCmd)
	deleteCmd.Flags().String("force", "", "force the deletion") // intentionally a string flag, not boolean, so that the user has to make that much extra effort to type
	deleteCmd.Flags().Bool("with-variants", false, "also delete variants if the TM has variants")
}

func executeDelete(cmd *cobra.Command, args []string) {
	force := cmd.Flag("force").Value.String()
	withVariants, err := cmd.Flags().GetBool("with-variants")
	if err != nil {
		cli.Stderrf("Could not parse --with-variants flag: %v", err)
		os.Exit(1)
	}

	spec := RepoSpecFromFlags(cmd)

	if force != "true" {
		cli.Stderrf("Cannot delete a TM unless --force is set to \"true\"")
		os.Exit(1)
	}

	err = cli.Delete(context.Background(), spec, args[0], commands.DeleteOptions{WithVariants: withVariants})
	if err != nil {
		cli.Stderrf("delete failed")
		os.Exit(1)
	}
}
