package nats

import (
	"context"
	"errors"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"google.golang.org/protobuf/proto"
)

//MarshalerFunc marshal struct to bytes.
type MarshalerFunc = func(v interface{}) ([]byte, error)

// Publisher implements a eventbus.Publisher interface.
type Publisher struct {
	dataMarshaler MarshalerFunc
	conn          *nats.Conn
}

// NewPublisher creates new publisher with proto marshaller.
func NewPublisher(config Config, opts ...Option) (*Publisher, error) {
	for _, o := range opts {
		config = o(config)
	}

	p, err := newPublisher(config.URL, utils.Marshal, config.Options...)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// NewPublisher creates a publisher.
func newPublisher(url string, eventMarshaler MarshalerFunc, options ...nats.Option) (*Publisher, error) {
	conn, err := nats.Connect(url, options...)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to server: %w", err)
	}

	return &Publisher{
		dataMarshaler: eventMarshaler,
		conn:          conn,
	}, nil
}

// Publish publishes an event to topics.
func (p *Publisher) Publish(ctx context.Context, topics []string, groupId, aggregateId string, event eventbus.Event) error {
	data, err := p.dataMarshaler(event)
	if err != nil {
		return errors.New("could not marshal data for event: " + err.Error())
	}

	e := pb.Event{
		EventType:   event.EventType(),
		Data:        data,
		Version:     event.Version(),
		GroupId:     groupId,
		AggregateId: aggregateId,
	}

	eData, err := proto.Marshal(&e)
	if err != nil {
		return errors.New("could not marshal event: " + err.Error())
	}

	var errors []error
	for _, t := range topics {
		err := p.conn.Publish(t, eData)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot publish events: %v", errors)
	}

	return nil
}

// Close close connection to nats
func (p *Publisher) Close() {
	p.conn.Close()
}
