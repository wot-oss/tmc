package http

import (
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/jwt"
	"github.com/web-of-things-open-source/tm-catalog-cli/internal/app/http/server"
)

func CollectMiddlewares(opts ServerOptions) []server.MiddlewareFunc {
	var mws []server.MiddlewareFunc
	if opts.JWTValidation == true {
		mws = append(mws, jwt.JWTValidationMiddleware)
	}
	return mws
}
