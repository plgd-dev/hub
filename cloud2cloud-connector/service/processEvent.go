package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/events"
)

func (rh *RequestHandler) notifyLinkedAccount(r *http.Request) (int, error) {
	header, err := events.ParseEventHeader(r)
	if err != nil {
		return http.StatusGone, err
	}

	b := bytes.NewBuffer(make([]byte, 0, 1024))
	_, err = b.ReadFrom(r.Body)
	if err != nil {
		return http.StatusGone, err
	}

	return rh.subManager.handleEvent(r.Context(), header, b.Bytes())
}

func (rh *RequestHandler) ProcessEvent(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.notifyLinkedAccount(r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot notify linked accounts: %w", err), statusCode, w)
	}
}
