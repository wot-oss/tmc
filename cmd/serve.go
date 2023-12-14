package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve TMs in catalog",
	Long:  `Serve TMs in catalog`,
	Args:  cobra.MaximumNArgs(0),
	Run:   serve,
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("host", "", "0.0.0.0", "serve with this host name")
	serveCmd.Flags().StringP("port", "", "8080", "serve with this port")
	serveCmd.Flags().StringP("contextRoot", "", "", "define additional context root path")
}

func serve(cmd *cobra.Command, args []string) {
	host := cmd.Flag("host").Value.String()
	port := cmd.Flag("port").Value.String()
	cr := cmd.Flag("contextRoot").Value.String()

	err := cli.Serve(host, port, cr)
	if err != nil {
		cli.Stderrf("serve failed")
		os.Exit(1)
	}
}
