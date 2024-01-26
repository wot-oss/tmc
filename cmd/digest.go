package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var digestCmd = &cobra.Command{
	Use:   "digest <FILENAME>",
	Short: "calculate version digest of a TM file",
	Long:  `calculate version digest of a TM file. The file must be json and contain a json object, but it is not validated as a TM`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.CalcFileDigest(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	//RootCmd.AddCommand(digestCmd) // fixme: discuss whether this command should stay
}
