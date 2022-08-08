package service

import (
	"net/http"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func jsonResponseWriterEncoder(w http.ResponseWriter, v interface{}, status int) error {
	if v == nil {
		return nil
	}
	w.Header().Set(events.ContentTypeKey, message.AppJSON.String())
	w.WriteHeader(status)
	return json.WriteTo(w, v)
}
