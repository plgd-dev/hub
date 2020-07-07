package service

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

func (rh *RequestHandler) deleteLinkedCloud(w http.ResponseWriter, r *http.Request) (int, error) {
	linkedCloudID, _ := mux.Vars(r)[cloudIDKey]
	cloud, err := rh.store.PullOutCloud(r.Context(), linkedCloudID)
	if err != nil {
		return http.StatusBadRequest, err
	}
	var wg sync.WaitGroup
	cloud.linkedAccounts.Range(func(_, accountI interface{}) bool {
		account := accountI.(*LinkedAccountData)
		wg.Add(1)
		go func() {
			defer wg.Done()
			cancelLinkedAccountSubscription(r.Context(), cloud.linkedCloud, account)
		}()
		return true
	})
	wg.Wait()
	return http.StatusOK, nil
}

func (rh *RequestHandler) DeleteLinkedCloud(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.deleteLinkedCloud(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot delete linked cloud: %v", err), statusCode, w)
	}
}
