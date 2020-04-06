package service

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func (rh *RequestHandler) deleteLinkedCloud(w http.ResponseWriter, r *http.Request) (int, error) {
	linkedCloudID, _ := mux.Vars(r)[linkedCloudIdKey]
	err := rh.store.RemoveLinkedCloud(r.Context(), linkedCloudID)
	if err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) DeleteLinkedCloud(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.deleteLinkedCloud(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot delete linked cloud: %v", err), statusCode, w)
	}
}
