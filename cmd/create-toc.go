package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/src/toc"
)

var createTOCCmd = &cobra.Command{
	Use:   "create-toc DIRECTORY",
	Short: "Creates a Table of Contents",
	Long:  "Creates a Table of Contents listing all paths to Thing Model files. Used for simple search functionality.",
	Run:   executeCreateTOC,
}

func init() {
	rootCmd.AddCommand(createTOCCmd)
	createTOCCmd.Flags().StringP("remote", "r", "", "use named remote instead of default")
}

func executeCreateTOC(cmd *cobra.Command, args []string) {
	var log = slog.Default()

	log.Debug("creating toc", "args", args)
	remoteName := cmd.Flag("remote").Value.String()

	toc.Create(remoteName)
}
