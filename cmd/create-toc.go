package cmd

import (
	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/src/toc"
)

var createTOCCmd = &cobra.Command{
	Use:   "create-toc DIRECTORY",
	Short: "Creates a Table of Contents",
	Long:  "Creates a Table of Contents listing all paths to Thing Model files. Used for simple search functionality.",
	Args:  cobra.ExactArgs(1),
	Run:   executeCreateTOC,
}

func init() {
	rootCmd.AddCommand(createTOCCmd)
	createTOCCmd.Flags().StringP("catalog-url", "c", "", "use only the catalog at the provided catalog URL")
}

func executeCreateTOC(cmd *cobra.Command, args []string) {

	toc.Create(args[0])
}
