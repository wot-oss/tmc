package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/cors"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/jwt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

//go:embed banner.txt
var banner string

type ServeOptions struct {
	UrlCtxRoot string
	cors.CORSOptions
	jwt.JWTValidationOpts
	JWTValidation bool
}

func Serve(host, port string, opts ServeOptions, repo, pushTarget remotes.RepoSpec) error {
	defer func() {
		if r := recover(); r != nil {
			Stderrf("could not start server:")
			Stderrf(fmt.Sprint(r))
		}
	}()
	err := validateContextRoot(opts.UrlCtxRoot)
	if err != nil {
		Stderrf(err.Error())
		return err
	}
	_, err = remotes.DefaultManager().Get(pushTarget)
	if err != nil {
		if errors.Is(err, remotes.ErrAmbiguous) {
			Stderrf("must specify target for push with --pushTarget when there are multiple remotes configured")
		} else if errors.Is(err, remotes.ErrRemoteNotFound) {
			Stderrf("invalid --pushTarget: %v", err)
		} else {
			Stderrf(err.Error())
		}
		return err
	}

	// create an instance of our handler (server interface)
	handlerService, err := http.NewDefaultHandlerService(remotes.DefaultManager(), repo, pushTarget)
	if err != nil {
		Stderrf("Could not start tm-catalog server on %s:%s, %v\n", host, port, err)
		return err
	}

	handler := http.NewTmcHandler(
		handlerService,
		http.TmcHandlerOptions{
			UrlContextRoot: opts.UrlCtxRoot,
		})

	// collect Middlewares for the main http handler
	var mws = getMiddlewares(opts)
	// create a http handler
	httpHandler := http.NewHttpHandler(handler, mws)
	// protect main handler with CORS
	httpHandler = cors.Protect(httpHandler, opts.CORSOptions)

	s := &nethttp.Server{
		Handler: httpHandler,
		Addr:    net.JoinHostPort(host, port),
	}

	// valid configuration, we can print the banner and start the server
	fmt.Println(banner)
	fmt.Printf("Version of tm-catalog-cli: %s\n", TmcVersion)
	fmt.Printf("Starting tm-catalog server on %s:%s\n", host, port)

	// start server
	err = s.ListenAndServe()
	if err != nil {
		Stderrf("Could not start tm-catalog server on %s:%s, %v\n", host, port, err)
		return err
	}

	return nil
}

func validateContextRoot(ctxRoot string) error {
	vCtxRoot, _ := url.JoinPath("/", ctxRoot)
	_, err := url.ParseRequestURI(vCtxRoot)
	if err != nil {
		return fmt.Errorf("invalid urlContextRoot: %s", ctxRoot)
	}
	return nil
}

func getMiddlewares(opts ServeOptions) []server.MiddlewareFunc {
	var mws []server.MiddlewareFunc
	if opts.JWTValidation == true {
		mws = append(mws, jwt.GetMiddleware(opts.JWTValidationOpts))
	}
	return mws
}
