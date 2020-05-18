package service

import (
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/kit/codec/json"
)

func jsonResponseWriterEncoder(w http.ResponseWriter, v interface{}, status int) error {
	if v == nil {
		return nil
	}
	w.Header().Set(events.ContentTypeKey, message.AppJSON.String())
	w.WriteHeader(status)
	return json.WriteTo(w, v)
}
