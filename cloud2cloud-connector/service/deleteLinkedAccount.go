package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/kit/log"

	"github.com/gorilla/mux"
)

func cancelLinkedAccountSubscription(ctx context.Context, cloud store.LinkedCloud, linkedAccount *LinkedAccountData) {
	var wg sync.WaitGroup
	if linkedAccount.isSubscribed {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cancelDevicesSubscription(ctx, linkedAccount.linkedAccount, cloud, linkedAccount.subscription.ID)
			if err != nil {
				log.Error(err)
			}
		}()
	}
	linkedAccount.devices.Range(func(_, deviceI interface{}) bool {
		device := deviceI.(*DeviceData)
		if device.isSubscribed {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := cancelDeviceSubscription(ctx, linkedAccount.linkedAccount, cloud, device.subscription.DeviceID, device.subscription.ID)
				if err != nil {
					log.Error(err)
				}
			}()
		}
		device.resources.Range(func(_, resourceI interface{}) bool {
			resource := resourceI.(*ResourceData)
			if resource.isSubscribed {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := cancelResourceSubscription(ctx, linkedAccount.linkedAccount, cloud, resource.subscription.DeviceID, resource.subscription.Href, resource.subscription.ID)
					if err != nil {
						log.Error(err)
					}
				}()
			}
			return true
		})
		return true
	})

	wg.Wait()
}

func (rh *RequestHandler) deleteLinkedAccount(w http.ResponseWriter, r *http.Request) (int, error) {
	vars := mux.Vars(r)
	cloudID := vars[cloudIDKey]
	accountID := vars[accountIDKey]

	linkedAccount, err := rh.store.PullOutLinkedAccount(r.Context(), cloudID, accountID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot load linked account: %w", err)
	}
	cloud, ok := rh.store.LoadCloud(cloudID)
	if !ok {
		return http.StatusOK, nil
	}
	cancelLinkedAccountSubscription(r.Context(), cloud, linkedAccount)

	return http.StatusOK, nil
}

func (rh *RequestHandler) DeleteLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.deleteLinkedAccount(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot delete linked accounts: %w", err), statusCode, w)
	}
}
