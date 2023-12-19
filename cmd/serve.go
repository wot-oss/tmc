package cmd

import (
	"os"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var serveCmd = &cobra.Command{
	Use:   "serve [--remote <remote-name>]",
	Short: "Serve TMs in catalog",
	Long: `Serve TMs in catalog.
If there are multiple remotes configured, --remote flag must be specified to define the target remote for push operations`,
	Args: cobra.MaximumNArgs(0),
	Run:  serve,
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("host", "", "0.0.0.0", "serve with this host name")
	serveCmd.Flags().StringP("port", "", "8080", "serve with this port")
	serveCmd.Flags().StringP("remote", "r", "", "name of the remote target for push")
	serveCmd.Flags().StringP("urlContextRoot", "", "",
		"define additional URL context root path to be considered in hypermedia links,\ncan also be set via environment variable TMC_URLCONTEXTROOT")
	_ = viper.BindPFlag(config.KeyUrlContextRoot, serveCmd.Flags().Lookup("urlContextRoot"))
}

func serve(cmd *cobra.Command, args []string) {
	host := cmd.Flag("host").Value.String()
	port := cmd.Flag("port").Value.String()
	remote := cmd.Flag("remote").Value.String()
	urlCtxRoot := viper.GetString(config.KeyUrlContextRoot)

	err := cli.Serve(host, port, urlCtxRoot, remote)
	if err != nil {
		cli.Stderrf("serve failed")
		os.Exit(1)
	}
}
