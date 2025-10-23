package cli

import (
	"context"
	_ "embed"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"

	"github.com/wot-oss/tmc/internal/app/http/cors"
	"github.com/wot-oss/tmc/internal/config"
	"github.com/wot-oss/tmc/internal/model"
	"github.com/wot-oss/tmc/internal/repos"
	"github.com/wot-oss/tmc/internal/utils"

	"github.com/wot-oss/tmc/internal/app/http/jwt"
	"github.com/wot-oss/tmc/internal/app/http/server"

	"github.com/wot-oss/tmc/internal/app/http"
)

//go:embed banner.txt
var banner string

type ServeOptions struct {
	UrlCtxRoot string
	cors.CORSOptions
	jwt.JWTValidationOpts
	JWTValidation bool
}

func Serve(host, port string, opts ServeOptions, repo model.RepoSpec) error {
	var currRepo string
	log := utils.GetLogger(context.Background(), "cli.Serve")
	defer func() {
		if r := recover(); r != nil {
			e := fmt.Errorf("panic: could not start tmc server: %v", r).Error()
			log.Error(e)
			Stderrf(e)
		}
	}()
	err := validateContextRoot(opts.UrlCtxRoot)
	if err != nil {
		Stderrf(err.Error())
		log.Error(err.Error())
		return err
	}

	httpHandler, err := createHttpHandler(repo, opts)
	if err != nil {
		err = fmt.Errorf("Could not start tm catalog server on %s:%s, %v\n", host, port, err)
		Stderrf(err.Error())
		log.Error(err.Error())
		return err
	}
	// protect main handler with CORS
	httpHandler = cors.Protect(httpHandler, opts.CORSOptions)

	s := &nethttp.Server{
		Handler: httpHandler,
		Addr:    net.JoinHostPort(host, port),
	}

	// valid configuration, we can print the banner and start the server
	repos, _ := repos.ReadConfig()
	if repo.Dir() != "" || repo.RepoName() != "" {
		currRepo = "from command line flags: "
		dir := repo.Dir()
		if dir != "" {
			currRepo += dir
		} else {
			currRepo += repo.RepoName()
		}
	} else if len(repos) > 0 {
		currRepo = "from config file: "
		for r := range repos {
			currRepo += r + " "
		}
	} else {
		err = fmt.Errorf("could not start tm catalog server on %s:%s, there's no catalog to serve", host, port)
		Stderrf(err.Error())
		log.Error(err.Error())
		return err
	}
	fmt.Println(banner)
	verMsg := fmt.Sprintf("Version of tmc: %s", utils.GetTmcVersion())
	startMsg := fmt.Sprintf("Starting tmc server on %s:%s serving catalog(s) %s", host, port, currRepo)
	fmt.Println(verMsg)
	fmt.Println(startMsg)
	fmt.Println()
	log.Info(verMsg)
	log.Info(startMsg)

	// start server
	err = s.ListenAndServe()
	if err != nil {
		err = fmt.Errorf("Could not start tm catalog server on %s:%s, %v\n", host, port, err)
		Stderrf(err.Error())
		log.Error(err.Error())
		return err
	}

	return nil
}

func createHttpHandler(repo model.RepoSpec, opts ServeOptions) (nethttp.Handler, error) {
	// create an instance of our handler (server interface)
	handlerService, err := http.NewDefaultHandlerService(repo)
	if err != nil {
		return nil, err
	}

	handler := http.NewTmcHandler(
		handlerService,
		http.TmcHandlerOptions{
			UrlContextRoot: opts.UrlCtxRoot,
			WhitelistPath:  config.WhitelistPath,
		})

	// collect Middlewares for the main http handler
	var mws = getMiddlewares(opts)
	// create a http handler
	httpHandler := http.NewHttpHandler(handler, mws)
	return httpHandler, nil
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
	mws = append(mws, http.WithLogAfterRequestProcessing)
	mws = append(mws, http.WithRequestLogger)
	if opts.JWTValidation == true {
		mws = append(mws, jwt.GetMiddleware(opts.JWTValidationOpts))
	}
	return mws
}
