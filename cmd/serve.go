package cmd

import (
	"os"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"

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
	serveCmd.Flags().StringP("urlContextRoot", "", "", "define additional URL context root path to be considered in hypermedia links")
	_ = viper.BindPFlag(config.KeyUrlContextRoot, serveCmd.Flags().Lookup("urlContextRoot"))
}

func serve(cmd *cobra.Command, args []string) {
	host := cmd.Flag("host").Value.String()
	port := cmd.Flag("port").Value.String()
	urlCtxRoot := viper.GetString(config.KeyUrlContextRoot)

	err := cli.Serve(host, port, urlCtxRoot)
	if err != nil {
		cli.Stderrf("serve failed")
		os.Exit(1)
	}
}
