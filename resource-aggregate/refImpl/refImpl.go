package refImpl

import (
	"context"
	"encoding/json"
	"fmt"

	natsio "github.com/nats-io/nats.go"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/jetstream"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager"
)

type Config struct {
	Service   service.Config
	Nats      nats.Config        `envconfig:"NATS"`
	JetStream jetstream.Config   `envconfig:"JETSTREAM"`
	Listen    certManager.Config `envconfig:"LISTEN"`
	Dial      certManager.Config `envconfig:"DIAL"`
	Log       log.Config         `envconfig:"LOG"`
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

type RefImpl struct {
	eventstore        *jetstream.EventStore
	service           *service.Server
	publisher         *nats.Publisher
	clientCertManager certManager.CertManager
	serverCertManager certManager.CertManager
}

func Init(config Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)

	clientCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create client cert manager %w", err)
	}
	tlsConfig := clientCertManager.GetClientTLSConfig()
	config.JetStream.Options = append(config.JetStream.Options, natsio.Secure(tlsConfig))

	eventstore, err := jetstream.NewEventStore(config.JetStream, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb eventstore %w", err)
	}
	publisher, err := nats.NewPublisher(config.Nats, nats.WithTLS(tlsConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create kafka publisher %w", err)
	}

	serverCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create server cert manager %w", err)
	}

	log.Info(config.String())

	return &RefImpl{
		eventstore:        eventstore,
		service:           service.New(config.Service, logger, clientCertManager, serverCertManager, eventstore, publisher),
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
