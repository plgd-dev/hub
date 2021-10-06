package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/v2/resource-aggregate/cqrs/eventbus/pb"
	"google.golang.org/protobuf/proto"
)

//MarshalerFunc marshal struct to bytes.
type MarshalerFunc = func(v interface{}) ([]byte, error)

// Publisher implements a eventbus.Publisher interface.
type Publisher struct {
	dataMarshaler MarshalerFunc
	conn          *nats.Conn
	closeFunc     []func()
	publish       func(subj string, data []byte) error
}

func (p *Publisher) AddCloseFunc(f func()) {
	p.closeFunc = append(p.closeFunc, f)
}

type options struct {
	dataMarshaler MarshalerFunc
}

type Option interface {
	apply(o *options)
}

type MarshalerOpt struct {
	dataMarshaler MarshalerFunc
}

func (o MarshalerOpt) apply(opts *options) {
	opts.dataMarshaler = o.dataMarshaler
}

func WithMarshaler(dataMarshaler MarshalerFunc) MarshalerOpt {
	return MarshalerOpt{
		dataMarshaler: dataMarshaler,
	}
}

// Create publisher with existing NATS connection and proto marshaller
func New(conn *nats.Conn, jetstream bool, opts ...Option) (*Publisher, error) {
	cfg := options{
		dataMarshaler: json.Marshal,
	}
	for _, o := range opts {
		o.apply(&cfg)
	}

	publish := conn.Publish
	if jetstream {
		js, err := conn.JetStream()
		if err != nil {
			return nil, fmt.Errorf("cannot get jetstream context: %w", err)
		}
		publish = func(subj string, data []byte) error {
			_, err := js.Publish(subj, data)
			return err
		}
	}

	return &Publisher{
		dataMarshaler: cfg.dataMarshaler,
		conn:          conn,
		publish:       publish,
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
		err := p.PublishData(ctx, t, eData)
		if err != nil {
			errors = append(errors, err)
		}
	}

	err = p.Flush(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("cannot publish events: %v", errors)
	}

	return nil
}

func (p *Publisher) PublishData(ctx context.Context, subj string, data []byte) error {
	return p.publish(subj, data)
}

func (p *Publisher) Flush(ctx context.Context) error {
	return p.conn.Flush()
}

func (p *Publisher) Close() {
	for _, f := range p.closeFunc {
		f()
	}
}
