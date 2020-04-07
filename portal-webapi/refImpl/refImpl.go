package refImpl

import (
	"encoding/json"
	"fmt"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/kit/security/certManager"
	"github.com/go-ocf/ocf-cloud/portal-webapi/service"
)

type Config struct {
	Log     log.Config `envconfig:"LOG"`
	Service service.Config
	Dial    certManager.Config `envconfig:"DIAL"`
	Listen  certManager.Config `envconfig:"LISTEN"`
}

type RefImpl struct {
	server            *service.Server
	dialCertManager   certManager.CertManager
	listenCertManager certManager.CertManager
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)

	dialCertManager, err := certManager.NewCertManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %v", err)
	}
	listenCertManager, err := certManager.NewCertManager(config.Listen)
	if err != nil {
		return nil, fmt.Errorf("cannot create listen cert manager %v", err)
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
