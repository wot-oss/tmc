package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/cmd/completion"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var exportFilterFlags = cli.FilterFlags{}

var exportCmd = &cobra.Command{
	Use:   "export <NAME PATTERN>",
	Short: "Export multiple TMs from a catalog and, optionally, attachments",
	Long: `Exports one or more TMs from a catalog by name pattern, filters or search. 
The name can be a full name or a prefix consisting of complete path parts. 
E.g. 'MyCompany/BarTech' will not match 'MyCompany/BarTechCorp', but will match 'MyCompany/BarTech/BazLamp'.

Name pattern, filters and search can be combined to narrow down the result.`,
	Args:              cobra.MaximumNArgs(1),
	Run:               executeExport,
	ValidArgsFunction: completion.CompleteTMNames,
}

func init() {
	RootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringP("repo", "r", "", "Name of the repository to export from. Exports from all if omitted")
	_ = exportCmd.RegisterFlagCompletionFunc("repo", completion.CompleteRepoNames)
	exportCmd.Flags().StringP("directory", "d", "", "Use the specified directory as repository. This option allows directly using a directory as a local TM repository, forgoing creating a named repository.")
	_ = exportCmd.MarkFlagDirname("directory")
	exportCmd.Flags().StringP("output", "o", "", "output directory for saving exported TMs")
	_ = exportCmd.MarkFlagDirname("output")
	_ = exportCmd.MarkFlagRequired("output")
	exportCmd.Flags().StringVar(&exportFilterFlags.FilterAuthor, "filter.author", "", "filter TMs by one or more comma-separated authors")
	exportCmd.Flags().StringVar(&exportFilterFlags.FilterManufacturer, "filter.manufacturer", "", "filter TMs by one or more comma-separated manufacturers")
	exportCmd.Flags().StringVar(&exportFilterFlags.FilterMpn, "filter.mpn", "", "filter TMs by one or more comma-separated mpn (manufacturer part number)")
	exportCmd.Flags().StringVarP(&exportFilterFlags.Search, "search", "s", "", "search TMs by their content matching the search term")
	_ = exportCmd.MarkFlagRequired("output")
	exportCmd.Flags().BoolP("restore-id", "R", false, "restore the TMs' original external ids, if they had one")
	exportCmd.Flags().BoolP("with-attachments", "A", false, "also export attachments")
}

func executeExport(cmd *cobra.Command, args []string) {
	outputPath := cmd.Flag("output").Value.String()
	restoreId, _ := cmd.Flags().GetBool("restore-id")
	withAttachments, _ := cmd.Flags().GetBool("with-attachments")

	spec := RepoSpec(cmd)

	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	search := cli.CreateSearchParamsFromCLI(exportFilterFlags, name)
	err := cli.Export(context.Background(), spec, search, outputPath, restoreId, withAttachments)

	if err != nil {
		cli.Stderrf("export failed")
		os.Exit(1)
	}
}
