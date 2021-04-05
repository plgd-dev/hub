package http

import (
	"errors"

	"github.com/valyala/fasthttp"
)

// TextPlainContentType content type strings for text plain that is user for error.
var TextPlainContentType = "text/plain; charset=utf-8"

// ErrInternalServerError internal server error
var ErrInternalServerError = errors.New("internal server error")

// WriteErrorResponse sets the content type and encodes the error to the body.
func WriteErrorResponse(err error, resp *fasthttp.Response) {
	if err != nil {
		resp.Header.SetContentType(TextPlainContentType)
		resp.SetBody([]byte(err.Error()))
	}
}

// ReadErrorResponse validates the content type and decodes the body to the error.
func ReadErrorResponse(resp *fasthttp.Response) error {
	contentType := string(resp.Header.ContentType())
	switch contentType {
	case TextPlainContentType:
		errStr := string(resp.Body())
		if errStr == "" {
			return ErrInternalServerError
		}
		return errors.New(errStr)
	default:
		return ErrInternalServerError
	}
}
