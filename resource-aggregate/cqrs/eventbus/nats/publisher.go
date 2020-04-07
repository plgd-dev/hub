package nats

import (
	cqrsNats "github.com/go-ocf/cqrs/eventbus/nats"
	cqrsUtils "github.com/go-ocf/cloud/resource-aggregate/cqrs"
)

type Publisher struct {
	*cqrsNats.Publisher
}

// NewPublisher creates new publisher with proto marshaller.
func NewPublisher(config Config, opts ...Option) (*Publisher, error) {
	for _, o := range opts {
		config = o(config)
	}

	p, err := cqrsNats.NewPublisher(config.URL, cqrsUtils.Marshal, config.Options...)
	if err != nil {
		return nil, err
	}
	return &Publisher{
		p,
	}, nil
}

// Close closes the publisher.
func (p *Publisher) Close() {
	p.Publisher.Close()
}
