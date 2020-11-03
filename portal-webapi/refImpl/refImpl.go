package refImpl

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/portal-webapi/service"
	"github.com/plgd-dev/kit/log"
	"github.com/plgd-dev/kit/security/certificateManager"
)

type Config struct {
	Log     log.Config `envconfig:"LOG"`
	Service service.Config
	Dial    certificateManager.Config `envconfig:"DIAL"`
	Listen  certificateManager.Config `envconfig:"LISTEN"`
}

type RefImpl struct {
	server            *service.Server
	dialCertManager   *certificateManager.CertificateManager
	listenCertManager *certificateManager.CertificateManager
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

func Init(config Config) (*RefImpl, error) {
	log.Setup(config.Log)

	dialCertManager, err := certificateManager.NewCertificateManager(config.Dial)
	if err != nil {
		return nil, fmt.Errorf("cannot create dial cert manager %w", err)
	}
	listenCertManager, err := certificateManager.NewCertificateManager(config.Listen)
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
