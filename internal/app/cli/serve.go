package cli

import (
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

func Serve(host, port, urlCtxRoot string, repo, pushTarget remotes.RepoSpec) error {

	err := validateContextRoot(urlCtxRoot)
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

	// create an instance of a router and our handler
	r := http.NewRouter()

	handlerService, err := http.NewDefaultHandlerService(remotes.DefaultManager(), repo, pushTarget)
	if err != nil {
		Stderrf("Could not start tm-catalog server on %s:%s, %v\n", host, port, err)
		return err
	}
	handler := http.NewTmcHandler(
		handlerService,
		http.TmcHandlerOptions{
			UrlContextRoot: urlCtxRoot,
		})

	options := server.GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: http.HandleErrorResponse,
	}
	_ = server.HandlerWithOptions(handler, options)

	s := &nethttp.Server{
		Handler: r,
		Addr:    net.JoinHostPort(host, port),
	}

	fmt.Printf("Start tm-catalog server on %s:%s\n", host, port)
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
