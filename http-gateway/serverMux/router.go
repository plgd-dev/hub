package serverMux

import (
	"context"
	"net/http"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpJwt "github.com/plgd-dev/hub/v2/pkg/net/http/jwt"
	"google.golang.org/grpc/codes"
)

// NewRouter creates router with default middlewares
func NewRouter(queryCaseInsensitive map[string]string, authInterceptor pkgHttpJwt.Interceptor, opts ...pkgHttp.LogOpt) *router.Router {
	r := router.NewRouter()
	r.Use(pkgHttp.CreateLoggingMiddleware(opts...))
	r.Use(pkgHttp.CreateAuthMiddleware(authInterceptor, func(_ context.Context, w http.ResponseWriter, r *http.Request, err error) {
		WriteError(w, grpc.ForwardErrorf(codes.Unauthenticated, "cannot access to %v: %w", r.RequestURI, err))
	}))
	r.Use(pkgHttp.CreateMakeQueryCaseInsensitiveMiddleware(queryCaseInsensitive, opts...))
	r.Use(pkgHttp.CreateTrailSlashSuffixMiddleware(opts...))
	return r
}
