package serverMux

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
)

// New creates default server mux
func New(opts ...runtime.ServeMuxOption) *runtime.ServeMux {
	intOpts := make([]runtime.ServeMuxOption, 0, 3+len(opts))
	intOpts = append(intOpts, runtime.WithErrorHandler(ErrorHandler))
	intOpts = append(intOpts, runtime.WithMarshalerOption(pkgHttp.ApplicationProtoJsonContentType, NewJsonpbMarshaler()))
	intOpts = append(intOpts, runtime.WithMarshalerOption(runtime.MIMEWildcard, NewJsonMarshaler()))
	intOpts = append(intOpts, opts...)
	return runtime.NewServeMux(intOpts...)
}
