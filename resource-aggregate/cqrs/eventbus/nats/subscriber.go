package nats

import (
	cqrsUtils "github.com/plgd-dev/cloud/resource-aggregate/cqrs"
	cqrsEventBus "github.com/plgd-dev/cqrs/eventbus"
	cqrsNats "github.com/plgd-dev/cqrs/eventbus/nats"
)

type Subscriber struct {
	*cqrsNats.Subscriber
}

// NewSubscriber create new subscriber with proto unmarshaller.
func NewSubscriber(config Config, goroutinePoolGo cqrsEventBus.GoroutinePoolGoFunc, errFunc cqrsEventBus.ErrFunc, opts ...Option) (*Subscriber, error) {
	for _, o := range opts {
		config = o(config)
	}

	s, err := cqrsNats.NewSubscriber(config.URL, cqrsUtils.Unmarshal, goroutinePoolGo, errFunc, config.Options...)
	if err != nil {
		return nil, err
	}
	return &Subscriber{
		s,
	}, nil
}

// Close closes the publisher.
func (p *Subscriber) Close() {
	p.Subscriber.Close()
}
