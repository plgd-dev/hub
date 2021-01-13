package refImpl

import (
	"encoding/json"
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/service"
	"github.com/plgd-dev/kit/log"
)

type Config struct {
	Log              log.Config            	`envconfig:"LOG" long:"log"`
	Service          service.Config		   	`long:"apis"`
	Clients			 service.ClientsConfig  `long:"clients"`
}

type RefImpl struct {
	service           *service.Server
}

//String return string representation of Config
func (c Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return fmt.Sprintf("config: \n%v\n", string(b))
}

// Init creates reference implementation for coap-gateway with default authorization interceptor.
func Init(config Config) (*RefImpl, error) {

	log.Setup(config.Log)
	log.Info(config.String())

	return &RefImpl{
		service:           service.New(config.Service, config.Clients),
	}, nil
}

// Serve starts handling coap requests.
func (r *RefImpl) Serve() error {
	return r.service.Serve()
}

// Shutdown shutdowns the service.
func (r *RefImpl) Shutdown() error {
	err := r.service.Shutdown()
	/*r.dialCertManager.Close()
	if r.listenCertManager != nil {
		r.listenCertManager.Close()
	}*/
	return err
}
