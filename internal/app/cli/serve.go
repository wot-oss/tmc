package cli

import (
	"fmt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http"
	"net"
	nethttp "net/http"
)

func Serve(host, port string) error {

	// create an instance of a router and our handler
	r := http.NewRouter()
	handler := http.NewTmcHandler()

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
	err := s.ListenAndServe()
	if err != nil {
		Stderrf("Could not start tm-catalog server on %s:%s, %v\n", err)
		return err
	}

	return nil
}
