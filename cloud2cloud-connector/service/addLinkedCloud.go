package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/cloud2cloud-connector/events"
	"github.com/plgd-dev/hub/cloud2cloud-connector/store"
	"github.com/plgd-dev/kit/v2/codec/json"
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
		return http.StatusBadRequest, fmt.Errorf("cannot read body: %w", err)
	}
	var l store.LinkedCloud
	err = json.Decode(buffer.Bytes(), &l)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot decode body: %w", err)
	}
	uuid, err1 := uuid.NewRandom()
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
		logAndWriteErrorResponse(fmt.Errorf("cannot add linked cloud: %w", err), statusCode, w)
	}
}
