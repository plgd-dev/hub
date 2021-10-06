package client

import (
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/service"
	"google.golang.org/grpc"
)

type Client struct {
	service.ResourceAggregateClient
	subscriber eventbus.Subscriber
}

func New(cc grpc.ClientConnInterface, subscriber eventbus.Subscriber) *Client {
	return &Client{
		ResourceAggregateClient: service.NewResourceAggregateClient(cc),
		subscriber:              subscriber,
	}
}
