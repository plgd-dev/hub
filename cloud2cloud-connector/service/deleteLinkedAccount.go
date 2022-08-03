package service

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"go.opentelemetry.io/otel/trace"
)

func cancelLinkedAccountDevicesSubscription(ctx context.Context, traceProvider trace.TracerProvider, cloud store.LinkedCloud, linkedAccount *LinkedAccountData, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := cancelDevicesSubscription(ctx, traceProvider, linkedAccount.linkedAccount, cloud, linkedAccount.subscription.ID)
		if err != nil {
			log.Error(err)
		}
	}()
}

func cancelLinkedAccountDeviceSubscription(ctx context.Context, traceProvider trace.TracerProvider, cloud store.LinkedCloud, linkedAccount *LinkedAccountData, device *DeviceData, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := cancelDeviceSubscription(ctx, traceProvider, linkedAccount.linkedAccount, cloud, device.subscription.DeviceID, device.subscription.ID)
		if err != nil {
			log.Error(err)
		}
	}()
}

func cancelLinkedAccountResourceSubscription(ctx context.Context, traceProvider trace.TracerProvider, cloud store.LinkedCloud, linkedAccount *LinkedAccountData, resource *ResourceData, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cancelResourceSubscription(ctx, traceProvider, linkedAccount.linkedAccount, cloud, resource.subscription.DeviceID, resource.subscription.Href, resource.subscription.ID); err != nil {
			log.Error(err)
		}
	}()
}

func cancelLinkedAccountSubscription(ctx context.Context, traceProvider trace.TracerProvider, cloud store.LinkedCloud, linkedAccount *LinkedAccountData) {
	var wg sync.WaitGroup
	if linkedAccount.isSubscribed {
		cancelLinkedAccountDevicesSubscription(ctx, traceProvider, cloud, linkedAccount, &wg)
	}
	linkedAccount.devices.Range(func(_, deviceI interface{}) bool {
		device := deviceI.(*DeviceData)
		if device.isSubscribed {
			cancelLinkedAccountDeviceSubscription(ctx, traceProvider, cloud, linkedAccount, device, &wg)
		}
		device.resources.Range(func(_, resourceI interface{}) bool {
			resource := resourceI.(*ResourceData)
			if resource.isSubscribed {
				cancelLinkedAccountResourceSubscription(ctx, traceProvider, cloud, linkedAccount, resource, &wg)
			}
			return true
		})
		return true
	})

	wg.Wait()
}

func (rh *RequestHandler) deleteLinkedAccount(r *http.Request) (int, error) {
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
	cancelLinkedAccountSubscription(r.Context(), rh.tracerProvider, cloud, linkedAccount)

	return http.StatusOK, nil
}

func (rh *RequestHandler) DeleteLinkedAccount(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.deleteLinkedAccount(r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot delete linked accounts: %w", err), statusCode, w)
	}
}
