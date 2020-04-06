package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
)

type LinkedAccountsHandler struct {
	linkedAccounts []store.LinkedAccount
}

func (h *LinkedAccountsHandler) Handle(ctx context.Context, iter store.LinkedAccountIter) (err error) {
	var s store.LinkedAccount
	for iter.Next(ctx, &s) {
		h.linkedAccounts = append(h.linkedAccounts, s)
	}
	return iter.Err()
}

func (rh *RequestHandler) retrieveLinkedAccounts(w http.ResponseWriter, r *http.Request) (int, error) {
	var h LinkedAccountsHandler
	err := rh.store.LoadLinkedAccounts(r.Context(), store.Query{}, &h)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = writeJson(w, h.linkedAccounts)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveLinkedAccounts(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveLinkedAccounts(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve linked accounts: %v", err), statusCode, w)
	}
}
