package cmd

import (
	"errors"
	"os"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/cors"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/jwt"

	"github.com/spf13/viper"
	"github.com/web-of-things-open-source/tm-catalog-cli/cmd/completion"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/config"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/utils"

	"github.com/spf13/cobra"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/cli"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve TMs in catalog",
	Long: `Serve TMs in catalog.
A target for push operations must be specified with --pushTarget in case neither --repository nor --directory is given.
This may be omitted if there's exactly one remote configured`,
	Args: cobra.MaximumNArgs(0),
	Run:  serve,
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("host", "", "0.0.0.0", "Serve with this host name")
	serveCmd.Flags().StringP("port", "", "8080", "Serve with this port")
	serveCmd.Flags().StringP("remote", "r", "", "Name of the remote to serve")
	_ = serveCmd.RegisterFlagCompletionFunc("remote", completion.CompleteRemoteNames)
	serveCmd.Flags().StringP("directory", "d", "", "TM repository directory to serve")
	_ = serveCmd.MarkFlagDirname("directory")
	serveCmd.Flags().StringP("pushTarget", "t", "", "Name of the remote to use as target for push operations")
	_ = serveCmd.RegisterFlagCompletionFunc("pushTarget", completion.CompleteRemoteNames)
	serveCmd.Flags().String(config.KeyUrlContextRoot, "",
		"Define additional URL context root path to be considered in hypermedia links (env var TMC_URLCONTEXTROOT)")
	serveCmd.Flags().String(config.KeyCorsAllowedOrigins, "", "Set comma-separated list for CORS allowed origins (env var TMC_CORSALLOWEDORIGINS)")
	serveCmd.Flags().String(config.KeyCorsAllowedHeaders, "", "Set comma-separated list for CORS allowed headers (env var TMC_CORSALLOWEDHEADERS)")
	serveCmd.Flags().Bool(config.KeyCorsAllowCredentials, false, "set CORS allow credentials (env var TMC_CORSALLOWCREDENTIALS)")
	serveCmd.Flags().Int(config.KeyCorsMaxAge, 0, "set how long result of CORS preflight request can be cached in seconds (default 0, max 600) (env varTMC_CORSMAXAGE)")
	serveCmd.Flags().Bool(config.KeyJWTValidation, false, "If set to 'true', jwt tokens are used to grant access to the API (env var TMC_JWTVALIDATION)")
	serveCmd.Flags().String(config.KeyJWTServiceID, "", "If set to an identifier, value will be compared to 'aud' claim in validated JWT (env var TMC_JWTSERVICEID)")
	serveCmd.Flags().String(config.KeyJWKSURL, "", "URL to periodically fetch JSON Web Key Sets for token validation (env var TMC_JWKSURL)")

	_ = viper.BindPFlag(config.KeyUrlContextRoot, serveCmd.Flags().Lookup(config.KeyUrlContextRoot))
	_ = viper.BindPFlag(config.KeyCorsAllowedOrigins, serveCmd.Flags().Lookup(config.KeyCorsAllowedOrigins))
	_ = viper.BindPFlag(config.KeyCorsAllowedHeaders, serveCmd.Flags().Lookup(config.KeyCorsAllowedHeaders))
	_ = viper.BindPFlag(config.KeyCorsAllowCredentials, serveCmd.Flags().Lookup(config.KeyCorsAllowCredentials))
	_ = viper.BindPFlag(config.KeyCorsMaxAge, serveCmd.Flags().Lookup(config.KeyCorsMaxAge))
	_ = viper.BindPFlag(config.KeyJWTValidation, serveCmd.Flags().Lookup(config.KeyJWTValidation))
	_ = viper.BindPFlag(config.KeyJWTServiceID, serveCmd.Flags().Lookup(config.KeyJWTServiceID))
	_ = viper.BindPFlag(config.KeyJWKSURL, serveCmd.Flags().Lookup(config.KeyJWKSURL))
}

func serve(cmd *cobra.Command, args []string) {
	host := cmd.Flag("host").Value.String()
	port := cmd.Flag("port").Value.String()
	remote := cmd.Flag("remote").Value.String()
	dir := cmd.Flag("directory").Value.String()
	pushTarget := cmd.Flag("pushTarget").Value.String()
	spec, err := remotes.NewSpec(remote, dir)
	if errors.Is(err, remotes.ErrInvalidSpec) {
		cli.Stderrf("Invalid specification of repository to be served. --remote and --directory are mutually exclusive. Set at most one")
		os.Exit(1)
	}

	opts := getServeOptions()

	pushSpec := spec
	if remote == "" && dir == "" && pushTarget != "" {
		pushSpec = remotes.NewRemoteSpec(pushTarget)
	}
	err = cli.Serve(host, port, opts, spec, pushSpec)
	if err != nil {
		cli.Stderrf("serve failed")
		os.Exit(1)
	}
}

func getServeOptions() cli.ServeOptions {
	opts := cli.ServeOptions{}

	opts.UrlCtxRoot = viper.GetString(config.KeyUrlContextRoot)
	opts.JWTValidation = viper.GetBool(config.KeyJWTValidation)
	opts.JWTValidationOpts = getJWKSOptions()
	opts.CORSOptions = getCORSOptions()
	return opts
}

func getCORSOptions() cors.CORSOptions {
	opts := cors.CORSOptions{}

	opts.AddAllowedOrigins(utils.ParseAsList(viper.GetString(config.KeyCorsAllowedOrigins), cli.DefaultListSeparator, true)...)
	opts.AddAllowedHeaders(utils.ParseAsList(viper.GetString(config.KeyCorsAllowedHeaders), cli.DefaultListSeparator, true)...)
	opts.AllowCredentials(viper.GetBool(config.KeyCorsAllowCredentials))
	opts.MaxAge(viper.GetInt(config.KeyCorsMaxAge))

	return opts
}

func getJWKSOptions() jwt.JWTValidationOpts {
	opts := jwt.JWTValidationOpts{}
	opts.JWTServiceID = viper.GetString(config.KeyJWTServiceID)
	opts.JWKSURLString = viper.GetString(config.KeyJWKSURL)
	return opts
}
