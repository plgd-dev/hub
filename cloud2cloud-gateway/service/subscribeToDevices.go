package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	pbAS "github.com/go-ocf/cloud/authorization/pb"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/cloud/cloud2cloud-connector/events"
	oapiStore "github.com/go-ocf/cloud/cloud2cloud-connector/store"
	"github.com/go-ocf/cloud/cloud2cloud-gateway/store"
	"google.golang.org/grpc/status"
)

func (rh *RequestHandler) GetUsersDevices(ctx context.Context, r *http.Request) ([]string, error) {
	token, err := getAccessToken(r)
	if err != nil {
		return nil, fmt.Errorf("cannot get users devices: %w", err)
	}

	client, err := rh.asClient.GetUserDevices(kitNetGrpc.CtxWithToken(ctx, token), &pbAS.GetUserDevicesRequest{})
	if err != nil {
		return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
	}
	defer client.CloseSend()
	userDevices := make([]string, 0, 32)
	for {
		userDevice, err := client.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, status.Errorf(status.Convert(err).Code(), "cannot get users devices: %v", err)
		}
		userDevices = append(userDevices, userDevice.GetDeviceId())
	}
	return userDevices, nil
}

func (rh *RequestHandler) subscribeToDevices(w http.ResponseWriter, r *http.Request) (int, error) {
	userDevices, err := rh.GetUsersDevices(r.Context(), r)
	if err != nil {
		return http.StatusUnauthorized, err
	}
	if len(userDevices) == 0 {
		return http.StatusForbidden, fmt.Errorf("cannot get user devices: empty")
	}

	token, userID, err := parseAuth(r.Header.Get("Authorization"))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot parse authorization header: %w", err)
	}

	s, code, err := rh.makeSubscription(w, r, oapiStore.Type_Devices, userID, []events.EventType{
		events.EventType_DevicesRegistered,
		events.EventType_DevicesUnregistered,
		events.EventType_DevicesOnline,
		events.EventType_DevicesOffline,
	})
	if err != nil {
		return code, err
	}

	subscription := store.DevicesSubscription{
		Subscription: s,
		AccessToken:  token,
	}

	err = rh.store.SaveDevicesSubscription(r.Context(), subscription)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot save subscription: %w", err)
	}

	err = jsonResponseWriterEncoder(w, SubscriptionResponse{
		SubscriptionID: subscription.ID,
	})
	if err != nil {
		rh.store.PopSubscription(r.Context(), subscription.ID)
		return http.StatusBadRequest, fmt.Errorf("cannot write response: %w", err)
	}

	return http.StatusOK, nil
}

func (rh *RequestHandler) SubscribeToDevices(w http.ResponseWriter, r *http.Request) {
	statusCode, err := rh.subscribeToDevices(w, r)
	if err != nil {
		logAndWriteErrorResponse(fmt.Errorf("cannot subscribe to all devices: %w", err), statusCode, w)
	}
}
