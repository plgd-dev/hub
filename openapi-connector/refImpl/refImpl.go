package refImpl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/cloud/openapi-connector/service"
	storeMongodb "github.com/go-ocf/cloud/openapi-connector/store/mongodb"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/panjf2000/ants"
)

type Config struct {
	Log               log.Config     `envconfig:"LOG"`
	MongoDB           mongodb.Config `envconfig:"MONGODB"`
	Nats              nats.Config    `envconfig:"NATS"`
	Service           service.Config
	GoRoutinePoolSize int                `envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	Dial              certManager.Config `envconfig:"DIAL"`
	Listen            certManager.Config `envconfig:"LISTEN"`
	StoreMongoDB      storeMongodb.Config
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*service.Server, error) {
	log.Setup(config.Log)
	log.Info(config.String())
	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %v", err)
	}
	dialTLSConfig := dialCertManager.GetClientTLSConfig()

	pool, err := ants.NewPool(config.GoRoutinePoolSize)
	if err != nil {
		return nil, fmt.Errorf("cannot create goroutine pool: %v", err)
	}

	resourceEventstore, err := mongodb.NewEventStore(config.MongoDB, pool.Submit, mongodb.WithTLS(&dialTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource mongodb eventstore %v", err)
	}

	resourceSubscriber, err := nats.NewSubscriber(config.Nats, pool.Submit, func(err error) { log.Errorf("error occurs during receiving event: %v", err) }, nats.WithTLS(&dialTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create resource nats subscriber %v", err)
	}

	store, err := storeMongodb.NewStore(context.Background(), config.StoreMongoDB, storeMongodb.WithTLS(&dialTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb store %v", err)
	}

	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %v", err)
	}

	return service.New(config.Service, dialCertManager, listenCertManager, resourceEventstore, resourceSubscriber, store), nil
}
