package publisher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/plgd-dev/cloud/pkg/log"

	nats "github.com/nats-io/nats.go"
	cmClient "github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/pb"
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

// New creates new publisher with proto marshaller.
func New(config client.ConfigPublisher, logger log.Logger, opts ...Option) (*Publisher, error) {
	certManager, err := cmClient.New(config.TLS, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	config.Options = append(config.Options, nats.Secure(certManager.GetTLSConfig()))
	conn, err := nats.Connect(config.URL, config.Options...)
	if err != nil {
		certManager.Close()
		return nil, fmt.Errorf("cannot create nats connection: %w", err)
	}

	p, err := NewWithNATS(conn, config.JetStream, opts...)
	if err != nil {
		conn.Close()
		certManager.Close()
		return nil, err
	}
	p.AddCloseFunc(certManager.Close)
	return p, nil
}

// Create publisher with existing NATS connection
func NewWithNATS(conn *nats.Conn, jetstream bool, opts ...Option) (*Publisher, error) {
	cfg := options{
		dataMarshaler: json.Marshal,
	}
	for _, o := range opts {
		o.apply(&cfg)
	}
	p, err := newPublisher(conn, jetstream, cfg.dataMarshaler)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// NewPublisher creates a publisher.
func newPublisher(conn *nats.Conn, jetstream bool, eventMarshaler MarshalerFunc) (*Publisher, error) {
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
		dataMarshaler: eventMarshaler,
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

// Close close connection to nats
func (p *Publisher) Close() {
	p.conn.Close()
	for _, f := range p.closeFunc {
		f()
	}
}
