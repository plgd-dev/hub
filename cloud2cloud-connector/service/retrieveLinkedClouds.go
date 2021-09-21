package service

import (
	"fmt"
	"net/http"
)

func (rh *RequestHandler) retrieveLinkedClouds(w http.ResponseWriter, r *http.Request) (int, error) {
	if err := writeJson(w, rh.store.Dump()); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveLinkedClouds(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveLinkedClouds(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve linked clouds: %w", err), statusCode, w)
	}
}
