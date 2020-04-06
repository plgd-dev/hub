package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-ocf/ocf-cloud/openapi-connector/store"
)

type LinkedCloudsHandler struct {
	linkedClouds []store.LinkedCloud
}

func (h *LinkedCloudsHandler) Handle(ctx context.Context, iter store.LinkedCloudIter) (err error) {
	var s store.LinkedCloud
	for iter.Next(ctx, &s) {
		h.linkedClouds = append(h.linkedClouds, s)
	}
	return iter.Err()
}

func (rh *RequestHandler) retrieveLinkedClouds(w http.ResponseWriter, r *http.Request) (int, error) {
	var h LinkedCloudsHandler
	err := rh.store.LoadLinkedClouds(r.Context(), store.Query{}, &h)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	err = writeJson(w, h.linkedClouds)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (rh *RequestHandler) RetrieveLinkedClouds(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.retrieveLinkedClouds(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot retrieve linked clouds: %v", err), statusCode, w)
	}
}
