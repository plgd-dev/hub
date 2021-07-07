package service

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	exCodes "github.com/plgd-dev/cloud/grpc-gateway/pb/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

func errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	s := status.Convert(err)
	rec := httptest.NewRecorder()
	code := exCodes.Code(s.Code())
	switch code {
	case exCodes.MethodNotAllowed:
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, rec, r, err)
	default:
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		return
	}
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(code.ToHTTPCode())
	if _, err := w.Write(rec.Body.Bytes()); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}
