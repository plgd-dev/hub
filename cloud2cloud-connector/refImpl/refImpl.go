package refImpl

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/cloud2cloud-connector/service"
	storeMongodb "github.com/plgd-dev/cloud/cloud2cloud-connector/store/mongodb"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certificateManager"
)

type Config struct {
	Log              log.Config `envconfig:"LOG"`
	Service          service.Config
	Dial             certificateManager.Config `envconfig:"DIAL"`
	Listen           certificateManager.Config `envconfig:"LISTEN"`
	ListenWithoutTLS bool                      `envconfig:"LISTEN_WITHOUT_TLS"`
	StoreMongoDB     storeMongodb.Config
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

func Init(config Config) (*service.Server, error) {
	log.Setup(config.Log)
	log.Info(config.String())
	dialCertManager, err := certificateManager.NewCertificateManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %w", err)
	}
	dialTLSConfig := dialCertManager.GetClientTLSConfig()

	store, err := storeMongodb.NewStore(context.Background(), config.StoreMongoDB, storeMongodb.WithTLS(dialTLSConfig))
	if err != nil {
		return nil, fmt.Errorf("cannot create mongodb store %w", err)
	}

	var listenCertManager *certificateManager.CertificateManager
	if !config.ListenWithoutTLS {
		listenCertManager, err = certificateManager.NewCertificateManager(config.Listen)
		if err != nil {
			return nil, fmt.Errorf("cannot create listen cert manager %w", err)
		}
	}

	return service.New(config.Service, dialCertManager, listenCertManager, store), nil
}
