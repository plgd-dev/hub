package test

import (
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	"go.opentelemetry.io/otel/trace"
)

func NewClientAndPublisher(config client.ConfigPublisher, fileWatcher *fsnotify.Watcher, logger log.Logger, tp trace.TracerProvider, opts ...publisher.Option) (*client.Client, *publisher.Publisher, error) {
	c, err := client.New(config.Config, fileWatcher, logger, tp)
	if err != nil {
		return nil, nil, err
	}

	if config.LeadResourceType != nil && config.LeadResourceType.Enabled {
		opts = append(opts, publisher.WithLeadResourceType(config.LeadResourceType.GetCompiledRegexFilter(), config.LeadResourceType.Filter, config.LeadResourceType.UseUUID))
	}
	p, err := publisher.New(c.GetConn(), config.JetStream, opts...)
	if err != nil {
		c.Close()
		return nil, nil, err
	}

	return c, p, nil
}
