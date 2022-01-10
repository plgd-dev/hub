package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	exCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	"google.golang.org/grpc/status"
)

type errorResponseWrapperWriter struct {
	http.ResponseWriter
	code int
}

func newErrorResponseWrapperWriter(w http.ResponseWriter, code int) *errorResponseWrapperWriter {
	return &errorResponseWrapperWriter{
		ResponseWriter: w,
		code:           code,
	}
}

func (w *errorResponseWrapperWriter) WriteHeader(_ int) {
	w.ResponseWriter.WriteHeader(w.code)
}

func (w *errorResponseWrapperWriter) Flush() {
	f, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func errorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	var customStatus *runtime.HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}
	s := status.Convert(err)
	httpCode := exCodes.Code(s.Code()).ToHTTPCode() // set proper error code because we extended it
	if customStatus != nil {
		httpCode = customStatus.HTTPStatus
	}
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, newErrorResponseWrapperWriter(w, httpCode), r, err)
}
