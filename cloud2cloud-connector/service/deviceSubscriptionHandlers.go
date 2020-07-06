package service

import (
	"context"
	"fmt"

	"github.com/go-ocf/cloud/cloud2cloud-connector/store"
	grpcClient "github.com/go-ocf/cloud/grpc-gateway/client"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	pbRA "github.com/go-ocf/cloud/resource-aggregate/pb"
	"github.com/go-ocf/kit/log"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	kitSync "github.com/go-ocf/kit/sync"
)

type deviceSubscriptionHandlers struct {
	onResourceUpdatePending   func(ctx context.Context, val *pb.Event_ResourceUpdatePending) error
	onResourceRetrievePending func(ctx context.Context, val *pb.Event_ResourceRetrievePending) error
	onClose                   func()
	onError                   func(err error)
}

func (h *deviceSubscriptionHandlers) HandleResourceUpdatePending(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
	return h.onResourceUpdatePending(ctx, val)
}

func (h *deviceSubscriptionHandlers) HandleResourceRetrievePending(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
	return h.onResourceRetrievePending(ctx, val)
}

func (h *deviceSubscriptionHandlers) Error(err error) {
	h.onError(err)
}

func (h *deviceSubscriptionHandlers) OnClose() {
	h.onClose()
}

type deviceSubscription struct {
	*grpcClient.DeviceSubscription
}

type DevicesSubscription struct {
	data     *kitSync.Map // //[deviceID]*deviceSubscription
	rdClient pb.GrpcGatewayClient
	raClient pbRA.ResourceAggregateClient
}

func NewDevicesSubscription(rdClient pb.GrpcGatewayClient, raClient pbRA.ResourceAggregateClient) *DevicesSubscription {
	return &DevicesSubscription{
		data:     kitSync.NewMap(),
		rdClient: rdClient,
		raClient: raClient,
	}
}

func getKey(userID, deviceID string) string {
	return userID + "." + deviceID
}

func (c *DevicesSubscription) Add(deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var s deviceSubscription
	key := getKey(linkedAccount.UserID, deviceID)
	v, loaded := c.data.LoadOrStore(key, &s)
	if loaded {
		return nil
	}
	sub := v.(*deviceSubscription)
	h := deviceSubscriptionHandlers{
		onResourceUpdatePending: func(ctx context.Context, val *pb.Event_ResourceUpdatePending) error {
			return updateResource(ctx, c.raClient, val, linkedAccount, linkedCloud)
		},
		onResourceRetrievePending: func(ctx context.Context, val *pb.Event_ResourceRetrievePending) error {
			return retrieveResource(ctx, c.raClient, val, linkedAccount, linkedCloud)
		},
		onClose: func() {
			log.Debugf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", deviceID)
			c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		},
		onError: func(err error) {
			log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) ends with error: %v", deviceID, err)
			c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		},
	}
	devSub, err := grpcClient.NewDeviceSubscription(kitNetGrpc.CtxWithUserID(context.Background(), linkedAccount.UserID), deviceID, &h, &h, c.rdClient)
	if err != nil {
		c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		return fmt.Errorf("cannot create device %v pending subscription: %w", deviceID, err)
	}
	sub.DeviceSubscription = devSub
	return nil
}

func (c *DevicesSubscription) Delete(userID, deviceID string) error {
	key := getKey(userID, deviceID)
	v, ok := c.data.PullOut(key)
	if !ok {
		return nil
	}
	sub := v.(*deviceSubscription)
	if sub.DeviceSubscription == nil {
		return nil
	}
	wait, err := sub.DeviceSubscription.Cancel()
	if err != nil {
		return err
	}
	wait()
	return nil
}
