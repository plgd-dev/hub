package serverMux

import (
	"context"
	"net/http"

	router "github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	kitHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"google.golang.org/grpc/codes"
)

// NewRouter creates router with default middlewares
func NewRouter(queryCaseInsensitive map[string]string, authInterceptor kitHttp.Interceptor, opts ...kitHttp.LogOpt) *router.Router {
	r := router.NewRouter()
	r.Use(kitHttp.CreateLoggingMiddleware(opts...))
	r.Use(kitHttp.CreateAuthMiddleware(authInterceptor, func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		WriteError(w, grpc.ForwardErrorf(codes.Unauthenticated, "cannot access to %v: %w", r.RequestURI, err))
	}))
	r.Use(kitHttp.CreateMakeQueryCaseInsensitiveMiddleware(queryCaseInsensitive, opts...))
	r.Use(kitHttp.CreateTrailSlashSuffixMiddleware(opts...))
	return r
}
