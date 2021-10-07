package service

import (
	"net/http"

	"github.com/plgd-dev/hub/cloud2cloud-connector/events"

	"github.com/plgd-dev/kit/v2/codec/cbor"
)

func newCBORResponseWriterEncoder(contentType string) func(w http.ResponseWriter, v interface{}, status int) error {
	return func(w http.ResponseWriter, v interface{}, status int) error {
		if v == nil {
			return nil
		}
		w.Header().Set(events.ContentTypeKey, contentType)
		w.WriteHeader(status)
		return cbor.WriteTo(w, v)
	}
}
