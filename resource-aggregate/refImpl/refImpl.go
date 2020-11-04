package refImpl

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certificateManager"
)

type Config struct {
	Service service.Config
	Nats    nats.Config               `envconfig:"NATS"`
	MongoDB mongodb.Config            `envconfig:"MONGODB"`
	Listen  certificateManager.Config `envconfig:"LISTEN"`
	Dial    certificateManager.Config `envconfig:"DIAL"`
	Log     log.Config                `envconfig:"LOG"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

type RefImpl struct {
	eventstore        *mongodb.EventStore
	service           *service.Server
	publisher         *nats.Publisher
	clientCertManager *certificateManager.CertificateManager
	serverCertManager *certificateManager.CertificateManager
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)

	clientCertManager, err := certificateManager.NewCertificateManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create client cert manager %w", err)
	}
	tlsConfig := clientCertManager.GetClientTLSConfig()

	eventstore, err := mongodb.NewEventStore(config.MongoDB, nil, mongodb.WithTLS(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	publisher, err := nats.NewPublisher(config.Nats, nats.WithTLS(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create kafka publisher %w", err)
	}

	serverCertManager, err := certificateManager.NewCertificateManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}

	log.Info(config.String())

	return &RefImpl{
		eventstore:        eventstore,
		service:           service.New(config.Service, clientCertManager, serverCertManager, eventstore, publisher),
		publisher:         publisher,
		clientCertManager: clientCertManager,
		serverCertManager: serverCertManager,
	}, nil
}

func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

func (r *RefImpl) Shutdown() {
	r.service.Shutdown()
	r.eventstore.Close(context.Background())
	r.publisher.Close()
	r.clientCertManager.Close()
	r.serverCertManager.Close()
}
