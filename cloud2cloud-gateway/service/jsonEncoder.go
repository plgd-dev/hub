package service

import (
	"net/http"

	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/json"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
)

func jsonResponseWriterEncoder(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		return nil
	}
	w.Header().Set(events.ContentTypeKey, coap.AppJSON.String())
	return json.WriteTo(w, v)
}
