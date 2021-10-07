package test

import (
	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/publisher"
)

func NewClientAndPublisher(config client.ConfigPublisher, logger log.Logger, opts ...publisher.Option) (*client.Client, *publisher.Publisher, error) {
	c, err := client.New(config.Config, logger)
	if err != nil {
		return nil, nil, err
	}

	p, err := publisher.New(c.GetConn(), config.JetStream, opts...)
	if err != nil {
		c.Close()
		return nil, nil, err
	}

	return c, p, nil
}
