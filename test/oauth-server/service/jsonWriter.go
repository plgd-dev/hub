package service

import (
	"net/http"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/v2/codec/json"
)

const contentTypeHeaderKey = "Content-Type"

func jsonResponseWriter(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	w.Header().Set(contentTypeHeaderKey, message.AppJSON.String())
	return json.WriteTo(w, v)
}
