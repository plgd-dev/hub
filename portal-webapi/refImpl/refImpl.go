package refImpl

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/portal-webapi/service"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/plgd-dev/kit/security/certManager/server"
)

type Config struct {
	Log     log.Config `envconfig:"LOG"`
	Service service.Config
	Dial    client.Config `envconfig:"DIAL"`
	Listen  server.Config `envconfig:"LISTEN"`
}

type RefImpl struct {
	server            *service.Server
	dialCertManager   *client.CertManager
	listenCertManager *server.CertManager
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*RefImpl, error) {
	logger, err := log.NewLogger(config.Log)
	if err != nil {
		return nil, fmt.Errorf("cannot create logger %w", err)
	}
	log.Set(logger)
	log.Info(config.String())

	dialCertManager, err := client.New(config.Dial, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %w", err)
	}
	listenCertManager, err := server.New(config.Listen, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %w", err)
	}
	log.Info(config.String())

	return &RefImpl{
		server:            service.New(config.Service, dialCertManager, listenCertManager),
		dialCertManager:   dialCertManager,
		listenCertManager: listenCertManager,
	}, nil
}

func (r *RefImpl) Serve() error {
	return r.server.Serve()
}

func (r *RefImpl) Shutdown() {
	r.server.Shutdown()
	r.dialCertManager.Close()
	r.listenCertManager.Close()
}
