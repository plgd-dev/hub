package serverMux

import (
	"context"
	"errors"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	exCodes "github.com/plgd-dev/hub/v2/grpc-gateway/pb/codes"
	"google.golang.org/grpc/status"
)

type ErrorResponseWrapperWriter struct {
	http.ResponseWriter
	code int
}

func NewErrorResponseWrapperWriter(w http.ResponseWriter, code int) *ErrorResponseWrapperWriter {
	return &ErrorResponseWrapperWriter{
		ResponseWriter: w,
		code:           code,
	}
}

func (w *ErrorResponseWrapperWriter) WriteHeader(_ int) {
	w.ResponseWriter.WriteHeader(w.code)
}

func (w *ErrorResponseWrapperWriter) Flush() {
	f, ok := w.ResponseWriter.(http.Flusher)
	if ok {
		f.Flush()
	}
}

// ErrorHandler is a convenient HTTP error handler for grpc.
func ErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	var customStatus *runtime.HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}
	s := status.Convert(err)
	httpCode := exCodes.Code(s.Code()).ToHTTPCode() // set proper error code because we extended it
	if customStatus != nil {
		httpCode = customStatus.HTTPStatus
	}
	runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, NewErrorResponseWrapperWriter(w, httpCode), r, err)
}
