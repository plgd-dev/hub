package service

import (
	"fmt"
	"net/http"

	"github.com/go-ocf/cloud/openapi-connector/store"

	"github.com/gorilla/mux"
)

func (rh *RequestHandler) deleteLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	linkedAccountId, _ := mux.Vars(r)[linkedAccountIdKey]

	var h LinkedAccountHandler
	err := rh.store.LoadLinkedAccounts(r.Context(), store.Query{ID: linkedAccountId}, &h)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot load linked account: %v", err)
	}
	if !h.ok {
		return http.StatusBadRequest, fmt.Errorf("cannot load linked account: not found")
	}

	var errors []error

	err = rh.subManager.StopSubscriptions(r.Context(), h.linkedAccount)
	if err != nil {
		errors = append(errors, err)
	}

	err = rh.store.RemoveLinkedAccount(r.Context(), linkedAccountId)
	if err != nil {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		return http.StatusInternalServerError, fmt.Errorf("%v", errors)
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) DeleteLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.deleteLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot delete linked accounts: %v", err), statusCode, w)
	}
}
