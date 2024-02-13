package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/remotes"
)

//go:embed banner.txt
var banner string

func Serve(host, port, urlCtxRoot string, opts http.ServerOptions, repo, pushTarget remotes.RepoSpec) error {

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

	// create an instance of our handler (server interface)
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

	// create a http handler
	httpHandler := http.NewHttpHandler(handler)
	httpHandler = http.WithCORS(httpHandler, opts)

	s := &nethttp.Server{
		Handler: httpHandler,
		Addr:    net.JoinHostPort(host, port),
	}

	fmt.Println(banner)
	fmt.Printf("Version of tm-catalog-cli: %s\n", TmcVersion)
	fmt.Printf("Starting tm-catalog server on %s:%s\n", host, port)
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
