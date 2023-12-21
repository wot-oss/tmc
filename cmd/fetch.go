package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch (<NAME[:SEMVER|DIGEST]> | <TMID>) [--remote <remoteName>] [--output <filename>] [--with-path]",
	Short: "Fetches the TM by name or id",
	Long: `Fetches TM by name, optionally accepting semantic version or digest. 
--remote, -r
	The name of the remote repository is optional if there's only one remote configured
--output, -o
	Write output to a file instead of stdout. If <filename> is an existing folder, the content will be written into
	a file in that folder. If it's a file or no such file exists, the output is written into a file with given name
--with-path
	Create the folder structure defined by Thing Model ID under <filename>. --output value must be a folder when --with-path is given`,
	Args: cobra.ExactArgs(1),
	Run:  executeFetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringP("remote", "r", "", "name of the remote to fetch from")
	fetchCmd.Flags().StringP("output", "o", "", "write output to a file instead of stdout")
	fetchCmd.Flags().BoolP("with-path", "", false, "create folder structure under output")
}

func executeFetch(cmd *cobra.Command, args []string) {
	remoteName := cmd.Flag("remote").Value.String()
	outputFile := cmd.Flag("output").Value.String()
	withPath, err := cmd.Flags().GetBool("with-path")
	if err != nil {
		cli.Stderrf("invalid --with-path flag")
		os.Exit(1)
	}
	err = cli.Fetch(remoteName, args[0], outputFile, withPath)
	if err != nil {
		cli.Stderrf("fetch failed")
		os.Exit(1)
	}
}
