package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/hub/v2/cloud2cloud-connector/store"
	grpcClient "github.com/plgd-dev/hub/v2/grpc-gateway/client"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/service"
	kitSync "github.com/plgd-dev/kit/v2/sync"
	"go.opentelemetry.io/otel/trace"
)

const NOT_SUPPORTED_ERR = "not supported"

type deviceSubscriptionHandlers struct {
	onResourceUpdatePending   func(ctx context.Context, val *raEvents.ResourceUpdatePending) error
	onResourceRetrievePending func(ctx context.Context, val *raEvents.ResourceRetrievePending) error
	onError                   func(err error)
	getContext                func() (context.Context, context.CancelFunc)
}

func (h deviceSubscriptionHandlers) GetContext() (context.Context, context.CancelFunc) {
	return h.getContext()
}

func (h deviceSubscriptionHandlers) UpdateResource(ctx context.Context, event *raEvents.ResourceUpdatePending) error {
	return h.onResourceUpdatePending(ctx, event)
}

func (h deviceSubscriptionHandlers) RetrieveResource(ctx context.Context, event *raEvents.ResourceRetrievePending) error {
	return h.onResourceRetrievePending(ctx, event)
}

func (h deviceSubscriptionHandlers) DeleteResource(ctx context.Context, event *raEvents.ResourceDeletePending) error {
	return fmt.Errorf(NOT_SUPPORTED_ERR)
}

func (h deviceSubscriptionHandlers) CreateResource(ctx context.Context, event *raEvents.ResourceCreatePending) error {
	return fmt.Errorf(NOT_SUPPORTED_ERR)
}

func (h deviceSubscriptionHandlers) UpdateDeviceMetadata(ctx context.Context, event *raEvents.DeviceMetadataUpdatePending) error {
	return fmt.Errorf(NOT_SUPPORTED_ERR)
}

func (h deviceSubscriptionHandlers) OnDeviceSubscriberReconnectError(err error) {
	h.onError(err)
}

type DevicesSubscription struct {
	ctx               context.Context
	data              *kitSync.Map // //[deviceID]*deviceSubscription
	rdClient          pb.GrpcGatewayClient
	raClient          raService.ResourceAggregateClient
	subscriber        *subscriber.Subscriber
	reconnectInterval time.Duration
	tracerProvider    trace.TracerProvider
}

func NewDevicesSubscription(ctx context.Context, tracerProvider trace.TracerProvider, rdClient pb.GrpcGatewayClient, raClient raService.ResourceAggregateClient, subscriber *subscriber.Subscriber, reconnectInterval time.Duration) *DevicesSubscription {
	return &DevicesSubscription{
		data:              kitSync.NewMap(),
		rdClient:          rdClient,
		raClient:          raClient,
		reconnectInterval: reconnectInterval,
		ctx:               ctx,
		subscriber:        subscriber,
		tracerProvider:    tracerProvider,
	}
}

func getKey(userID, deviceID string) string {
	return userID + "." + deviceID
}

func (c *DevicesSubscription) Add(ctx context.Context, deviceID string, linkedAccount store.LinkedAccount, linkedCloud store.LinkedCloud) error {
	var s atomic.Value
	key := getKey(linkedAccount.UserID, deviceID)
	_, loaded := c.data.LoadOrStore(key, &s)
	if loaded {
		return nil
	}
	deviceSubscriber, err := grpcClient.NewDeviceSubscriber(func() (context.Context, context.CancelFunc) {
		return kitNetGrpc.CtxWithToken(c.ctx, linkedAccount.Data.Origin().AccessToken.String()), func() {
			// no-op
		}
	}, "*", deviceID, func() func() (when time.Time, err error) {
		var count uint64
		maxRand := c.reconnectInterval / 2
		if maxRand <= 0 {
			maxRand = time.Second * 10
		}
		return func() (when time.Time, err error) {
			count++
			r := rand.Int63n(int64(maxRand) / 2)
			next := time.Now().Add(c.reconnectInterval + time.Duration(r))
			log.Debugf("next iteration %v of retrying reconnect to grpc-client for deviceID %v will be at %v", count, deviceID, next)
			return next, nil
		}
	}, c.rdClient, c.subscriber, c.tracerProvider)
	if err != nil {
		c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		return fmt.Errorf("cannot create device %v pending subscription: %w", deviceID, err)
	}
	h := grpcClient.NewDeviceSubscriptionHandlers(deviceSubscriptionHandlers{
		onResourceUpdatePending: func(ctx context.Context, val *raEvents.ResourceUpdatePending) error {
			return updateResource(ctx, c.tracerProvider, c.raClient, val, linkedAccount, linkedCloud)
		},
		onResourceRetrievePending: func(ctx context.Context, val *raEvents.ResourceRetrievePending) error {
			return retrieveResource(ctx, c.tracerProvider, c.raClient, val, linkedAccount, linkedCloud)
		},
		onError: func(err error) {
			log.Errorf("device %v subscription(ResourceUpdatePending, ResourceRetrievePending) was closed", deviceID)
			c.data.Delete(getKey(linkedAccount.UserID, deviceID))
		},
	})
	deviceSubscriber.SubscribeToPendingCommands(ctx, h)

	s.Store(deviceSubscriber)
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
	sub := s.(*grpcClient.DeviceSubscriber)
	if sub == nil {
		return nil
	}
	return sub.Close()
}
