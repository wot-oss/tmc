package cmd

import (
	"os"

	"github.com/wot-oss/tmc/internal/app/http/cors"
	"github.com/wot-oss/tmc/internal/app/http/jwt"

	"github.com/spf13/viper"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/utils"

	"github.com/spf13/cobra"
	"github.com/wot-oss/tmc/internal/app/cli"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a REST API server",
	Long:  `Start a REST API server for accessing and manipulating the catalog`,
	Args:  cobra.MaximumNArgs(0),
	Run:   serve,
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("host", "", "0.0.0.0", "Serve with this host name")
	serveCmd.Flags().StringP("port", "", "8080", "Serve with this port")
	AddRepoConstraintFlags(serveCmd)
	serveCmd.Flags().String(config.KeyUrlContextRoot, "",
		"Define additional URL context root path to be considered in hypermedia links (env var TMC_URLCONTEXTROOT)")
	serveCmd.Flags().String(config.KeyCorsAllowedOrigins, "", "Set comma-separated list for CORS allowed origins (env var TMC_CORSALLOWEDORIGINS)")
	serveCmd.Flags().String(config.KeyCorsAllowedHeaders, "", "Set comma-separated list for CORS allowed headers (env var TMC_CORSALLOWEDHEADERS)")
	serveCmd.Flags().Bool(config.KeyCorsAllowCredentials, false, "set CORS allow credentials (env var TMC_CORSALLOWCREDENTIALS)")
	serveCmd.Flags().Int(config.KeyCorsMaxAge, 0, "set how long result of CORS preflight request can be cached in seconds (default 0, max 600) (env varTMC_CORSMAXAGE)")
	serveCmd.Flags().Bool(config.KeyJWTValidation, false, "If set to 'true', jwt tokens are used to grant access to the API (env var TMC_JWTVALIDATION)")
	serveCmd.Flags().String(config.KeyJWTServiceID, "", "If set to an identifier, value will be compared to 'aud' claim in validated JWT (env var TMC_JWTSERVICEID)")
	serveCmd.Flags().String(config.KeyJWTScopesPrefix, "", "If set to a prefix, scopes in validated JWT are expected to start with this prefix (env var TMC_JWTSCOPESPREFIX)")
	serveCmd.Flags().String(config.KeyJWKSURL, "", "URL to periodically fetch JSON Web Key Sets for token validation (env var TMC_JWKSURL)")
	serveCmd.Flags().String(config.KeyDefaultScopes, config.DefaultScopesPath, "path to the default scopes file")

	_ = viper.BindPFlag(config.KeyUrlContextRoot, serveCmd.Flags().Lookup(config.KeyUrlContextRoot))
	_ = viper.BindPFlag(config.KeyCorsAllowedOrigins, serveCmd.Flags().Lookup(config.KeyCorsAllowedOrigins))
	_ = viper.BindPFlag(config.KeyCorsAllowedHeaders, serveCmd.Flags().Lookup(config.KeyCorsAllowedHeaders))
	_ = viper.BindPFlag(config.KeyCorsAllowCredentials, serveCmd.Flags().Lookup(config.KeyCorsAllowCredentials))
	_ = viper.BindPFlag(config.KeyCorsMaxAge, serveCmd.Flags().Lookup(config.KeyCorsMaxAge))
	_ = viper.BindPFlag(config.KeyJWTValidation, serveCmd.Flags().Lookup(config.KeyJWTValidation))
	_ = viper.BindPFlag(config.KeyJWTServiceID, serveCmd.Flags().Lookup(config.KeyJWTServiceID))
	_ = viper.BindPFlag(config.KeyJWTScopesPrefix, serveCmd.Flags().Lookup(config.KeyJWTScopesPrefix))
	_ = viper.BindPFlag(config.KeyJWKSURL, serveCmd.Flags().Lookup(config.KeyJWKSURL))
	_ = viper.BindPFlag(config.KeyDefaultScopes, serveCmd.Flags().Lookup(config.KeyDefaultScopes))
}

func serve(cmd *cobra.Command, args []string) {
	host := cmd.Flag("host").Value.String()
	port := cmd.Flag("port").Value.String()
	spec := RepoSpecFromFlags(cmd)
	opts := getServeOptions()

	err := cli.Serve(host, port, opts, spec)
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
	opts.ScopesPrefix = viper.GetString(config.KeyJWTScopesPrefix)
	opts.WhitelistFile = viper.GetString(config.KeyDefaultScopes)
	return opts
}
