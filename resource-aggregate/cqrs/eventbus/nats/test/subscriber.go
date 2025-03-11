package test

import (
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"go.opentelemetry.io/otel/trace"
)

func NewClientAndSubscriber(config client.ConfigSubscriber, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...subscriber.Option) (*client.Client, *subscriber.Subscriber, error) {
	c, err := client.New(config.Config, fileWatcher, logger, tracerProvider)
	if err != nil {
		return nil, nil, err
	}

	p, err := subscriber.New(c.GetConn(), config.PendingLimits, config.LeadResourceType.IsEnabled(), logger, opts...)
	if err != nil {
		c.Close()
		return nil, nil, err
	}

	return c, p, nil
}
