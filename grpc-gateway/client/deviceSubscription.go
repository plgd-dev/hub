package client

import (
	"context"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
)

// DeviceSubscription subscription.
type DeviceSubscription struct {
	id  string
	sub *DeviceSubscriptions
}

// NewDeviceSubscription creates new devices subscriptions to listen events: resource published, resource unpublished.
// JWT token must be stored in context for grpc call.
func (c *Client) NewDeviceSubscription(ctx context.Context, deviceID string, handle SubscriptionHandler) (*DeviceSubscription, error) {
	return NewDeviceSubscription(ctx, deviceID, handle, handle, c.gateway)
}

// NewDeviceSubscription creates new devices subscriptions to listen events: resource published, resource unpublished.
// JWT token must be stored in context for grpc call.
func NewDeviceSubscription(ctx context.Context, deviceID string, closeErrorHandler SubscriptionHandler, handle interface{}, gwClient pb.GrpcGatewayClient) (*DeviceSubscription, error) {
	sub, err := NewDeviceSubscriptions(ctx, gwClient, closeErrorHandler.Error)
	if err != nil {
		return nil, err
	}
	s, err := sub.Subscribe(ctx, deviceID, closeErrorHandler, handle)
	if err != nil {
		wait, err1 := sub.Cancel()
		if err1 == nil {
			wait()
		}
		return nil, err
	}

	return &DeviceSubscription{
		id:  s.ID(),
		sub: sub,
	}, nil
}

// Cancel cancels subscription.
func (s *DeviceSubscription) Cancel() (wait func(), err error) {
	return s.sub.Cancel()
}

// ID returns subscription id.
func (s *DeviceSubscription) ID() string {
	return s.id
}

func ToDeviceSubscription(v interface{}, ok bool) (*DeviceSubscription, bool) {
	if !ok {
		return nil, false
	}
	if v == nil {
		return nil, false
	}
	s, ok := v.(*DeviceSubscription)
	return s, ok
}
