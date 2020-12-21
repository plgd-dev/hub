package jetstream

import (
	"context"

	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/nats.go"
)

type JsmConn struct {
	conn *nats.Conn
	mgr  *jsm.Manager
}

func connect(config Config) (*JsmConn, error) {
	nc, err := nats.Connect(config.URL, config.Options...)
	if err != nil {
		return nil, err
	}

	jsopts := []jsm.Option{}
	mgr, err := jsm.New(nc, jsopts...)
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &JsmConn{
		conn: nc,
		mgr:  mgr,
	}, nil
}

func (c *JsmConn) close() {
	c.conn.Close()
}

func (c *JsmConn) newConsumer(streamName string, opts ...jsm.ConsumerOption) (*jsm.Consumer, error) {
	return c.mgr.NewConsumer(streamName, opts...)
}

func (c *JsmConn) loadOrNewStreamFromDefault(name string, dflt api.StreamConfig, opts ...jsm.StreamOption) (stream *jsm.Stream, err error) {
	return c.mgr.LoadOrNewStreamFromDefault(name, dflt, opts...)
}

func (c *JsmConn) requestWithContext(ctx context.Context, subj string, data []byte) (*nats.Msg, error) {
	return c.conn.RequestWithContext(ctx, subj, data)
}

func (c *JsmConn) streamNames(filter *jsm.StreamNamesFilter) (names []string, err error) {
	return c.mgr.StreamNames(filter)
}

func (c *JsmConn) streams() (streams []*jsm.Stream, err error) {
	return c.mgr.Streams()
}
