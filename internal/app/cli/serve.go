package cli

import (
	"fmt"
	"net"
	nethttp "net/http"
	"net/url"

	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
)

func Serve(host, port string, urlCtxRoot string) error {

	err := validateContextRoot(urlCtxRoot)
	if err != nil {
		Stderrf(err.Error())
		return err
	}

	// create an instance of a router and our handler
	r := http.NewRouter()

	handler := http.NewTmcHandler(
		http.TmcHandlerOptions{
			UrlContextRoot: urlCtxRoot,
		})

	options := http.GorillaServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: http.HandleErrorResponse,
	}
	http.HandlerWithOptions(handler, options)

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
