package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/store"
	grpcClient "github.com/plgd-dev/cloud/grpc-gateway/client"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	raService "github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/log"
	kitSync "github.com/plgd-dev/kit/sync"
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

type DevicesSubscription struct {
	ctx               context.Context
	data              *kitSync.Map // //[deviceID]*deviceSubscription
	rdClient          pb.GrpcGatewayClient
	raClient          raService.ResourceAggregateClient
	reconnectInterval time.Duration
}

func NewDevicesSubscription(ctx context.Context, rdClient pb.GrpcGatewayClient, raClient raService.ResourceAggregateClient, reconnectInterval time.Duration) *DevicesSubscription {
	return &DevicesSubscription{
		data:              kitSync.NewMap(),
		rdClient:          rdClient,
		raClient:          raClient,
		reconnectInterval: reconnectInterval,
		ctx:               ctx,
	}
}

func getKey(userID, deviceID string) string {
	return userID + "." + deviceID
}

func (c *DevicesSubscription) Add(deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var s atomic.Value
	key := getKey(linkedAccount.UserID, deviceID)
	_, loaded := c.data.LoadOrStore(key, &s)
	if loaded {
		return nil
	}
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
			if !strings.Contains(err.Error(), "transport is closing") {
				return
			}
			for {
				log.Debugf("reconnect device %v subscription(ResourceUpdatePending, ResourceRetrievePending)")
				err = c.Add(deviceID, linkedAccount, linkedCloud)
				if err == nil {
					return
				}
				if !strings.Contains(err.Error(), "connection refused") {
					log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) cannot reconnect: %v", deviceID, err)
					return
				}
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(c.reconnectInterval):
				}
			}
		},
	}
	devSub, err := grpcClient.NewDeviceSubscription(kitNetGrpc.CtxWithOwner(c.ctx, linkedAccount.UserID), deviceID, &h, &h, c.rdClient)
	if err != nil {
		c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		return fmt.Errorf("cannot create device %v pending subscription: %w", deviceID, err)
	}
	s.Store(devSub)
	return nil
}

func (c *DevicesSubscription) Delete(userID, deviceID string) error {
	key := getKey(userID, deviceID)
	v, ok := c.data.PullOut(key)
	if !ok {
		return nil
	}
	s := v.(*atomic.Value).Load()
	if s == nil {
		return nil
	}
	sub := s.(*grpcClient.DeviceSubscription)
	if sub == nil {
		return nil
	}
	wait, err := sub.Cancel()
	if err != nil {
		return err
	}
	wait()
	return nil
}
