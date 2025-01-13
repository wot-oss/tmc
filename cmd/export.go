package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var exportFilterFlags = FilterFlags{}

var exportCmd = &cobra.Command{
	Use:   "export [<name-pattern>]",
	Short: "Export multiple TMs from a catalog and, optionally, their attachments",
	Long: `Export one or more TMs from a catalog by name pattern, filters or search. 

Accepts the same <name-pattern> and filter flags as list command.
Use list command with the same parameters to verify beforehand which TMs are going to be exported.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeExport,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(exportCmd)
	AddRepoConstraintFlags(exportCmd)
	AddOutputFormatFlag(exportCmd)
	exportCmd.Flags().StringP("output", "o", "", "output directory for saving exported TMs")
	_ = exportCmd.MarkFlagDirname("output")
	_ = exportCmd.MarkFlagRequired("output")
	AddTMFilterFlags(exportCmd, &exportFilterFlags)
	_ = exportCmd.MarkFlagRequired("output")
	exportCmd.Flags().BoolP("restore-id", "R", false, "restore the TMs' original external ids, if they had one")
	exportCmd.Flags().BoolP("with-attachments", "A", false, "also export attachments")
}

func executeExport(cmd *cobra.Command, args []string) {
	outputPath := cmd.Flag("output").Value.String()
	restoreId, _ := cmd.Flags().GetBool("restore-id")
	withAttachments, _ := cmd.Flags().GetBool("with-attachments")
	format := cmd.Flag("format").Value.String()

	spec := RepoSpecFromFlags(cmd)

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	search := CreateFiltersFromCLI(exportFilterFlags, name)
	err := cli.Export(context.Background(), spec, search, outputPath, restoreId, withAttachments, format)

	if err != nil {
		cli.Stderrf("export failed")
		os.Exit(1)
	}
}
