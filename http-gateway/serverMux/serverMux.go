package serverMux

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
)

// New creates default server mux
func New(opts ...runtime.ServeMuxOption) *runtime.ServeMux {
	intOpts := []runtime.ServeMuxOption{
		runtime.WithErrorHandler(ErrorHandler),
		runtime.WithMarshalerOption(pkgHttp.ApplicationProtoJsonContentType, NewJsonpbMarshaler()),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, NewJsonMarshaler()),
	}
	intOpts = append(intOpts, opts...)
	return runtime.NewServeMux(intOpts...)
}
