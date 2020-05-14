package service

import (
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"

	"github.com/go-ocf/kit/codec/cbor"
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
