package serverMux

import (
	"context"
	"net/http"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pktHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"google.golang.org/grpc/codes"
)

// NewRouter creates router with default middlewares
func NewRouter(queryCaseInsensitive map[string]string, authInterceptor pktHttp.Interceptor, opts ...pktHttp.LogOpt) *router.Router {
	r := router.NewRouter()
	r.Use(pktHttp.CreateLoggingMiddleware(opts...))
	r.Use(pktHttp.CreateAuthMiddleware(authInterceptor, func(_ context.Context, w http.ResponseWriter, r *http.Request, err error) {
		WriteError(w, grpc.ForwardErrorf(codes.Unauthenticated, "cannot access to %v: %w", r.RequestURI, err))
	}))
	r.Use(pktHttp.CreateMakeQueryCaseInsensitiveMiddleware(queryCaseInsensitive, opts...))
	r.Use(pktHttp.CreateTrailSlashSuffixMiddleware(opts...))
	return r
}
