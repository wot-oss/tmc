package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

// digestCmd is only intended for testing and is therefore commented out.
// If you need to just calculate version digests of some files, you can _temporarily_ remove comment in init()
var digestCmd = &cobra.Command{
	Use:   "digest <filename>",
	Short: "calculate version digest of a TM file",
	Long:  `calculate version digest of a TM file. The file must be json and contain a json object, but it is not validated`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := cli.CalcFileDigest(args[0])
		if err != nil {
			os.Exit(1)
		}
	},
}

func init() {
	// uncomment the next line if necessary for testing. Never commit it uncommented
	//RootCmd.AddCommand(digestCmd)
}
