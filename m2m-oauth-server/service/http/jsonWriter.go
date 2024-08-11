package http

import (
	"net/http"

	"github.com/plgd-dev/go-coap/v3/message"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func jsonResponseWriter(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	w.Header().Set(pkgHttp.ContentTypeHeaderKey, message.AppJSON.String())
	return json.WriteTo(w, v)
}
