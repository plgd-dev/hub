package serverMux

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/plgd-dev/hub/v2/http-gateway/uri"
)

// New creates default server mux
func New(opts ...runtime.ServeMuxOption) *runtime.ServeMux {
	intOpts := []runtime.ServeMuxOption{
		runtime.WithErrorHandler(ErrorHandler),
		runtime.WithMarshalerOption(uri.ApplicationProtoJsonContentType, NewJsonpbMarshaler()),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, NewJsonMarshaler()),
	}
	intOpts = append(intOpts, opts...)
	return runtime.NewServeMux(intOpts...)
}
