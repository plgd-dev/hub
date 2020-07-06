package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/gofrs/uuid"

	"github.com/go-ocf/kit/codec/json"
)

func writeJson(w http.ResponseWriter, v interface{}) error {
	data, err := json.Encode(v)
	if err != nil {
		return err
	}
	w.Header().Set(events.ContentTypeKey, "application/json")
	_, err = w.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (rh *RequestHandler) addLinkedCloud(w http.ResponseWriter, r *http.Request) (int, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err := buffer.ReadFrom(r.Body)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot read body: %v", err)
	}
	var l store.LinkedCloud
	err = json.Decode(buffer.Bytes(), &l)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot decode body: %v", err)
	}
	uuid, err1 := uuid.NewV4()
	if err1 != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot generate uuid %v", err1)
	}
	l.ID = uuid.String()
	l, _, err = rh.store.LoadOrCreateCloud(r.Context(), l)
	if err != nil {
		return http.StatusBadRequest, err
	}
	err = writeJson(w, l)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) AddLinkedCloud(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.addLinkedCloud(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked cloud: %v", err), statusCode, w)
	}
}
