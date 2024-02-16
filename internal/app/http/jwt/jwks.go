package jwt

import (
	"fmt"
	"net/url"
	"time"
)

// TODO(pedram): set this to 15 Min after testing
const MinIntervalDuration = 2 * time.Second

var jwksURL *url.URL

type JWKSOpts struct {
	JWKSInterval  time.Duration
	JWKSURLString string
}

func validateOptions(opts JWKSOpts) {
	// disallow intervals that are too short
	if opts.JWKSInterval < MinIntervalDuration {
		msg := "jwks fetch interval must be set to a minimum of 15 minutes: %s"
		msg = fmt.Sprintf(msg, opts.JWKSInterval)
		panic(msg)
	}

	// JWKS URL must be initialized
	var err error
	jwksURL, err = url.ParseRequestURI(opts.JWKSURLString)
	if err != nil {
		msg := fmt.Sprintf(`jwt validation activated, but jwks URL could not be initialized:
%s
Check documentation to "serve" to configure correctly
`, err)
		panic(msg)
	}
}

func StartJWKSFetch(opts JWKSOpts) {
	validateOptions(opts)
	go func() {
		ticker := time.NewTicker(opts.JWKSInterval)
		for {
			select {
			case <-ticker.C:
				fetchJWKS()
			}
		}
	}()
}

func fetchJWKS() {

}
