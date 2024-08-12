package service

import (
	"errors"
	"fmt"

	certAuthorityPb "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcGatewayPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	m2mOAuthServerPb "github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	snippetServicePb "github.com/plgd-dev/hub/v2/snippet-service/pb"
)

type Config struct {
	Log  log.Config `yaml:"log" json:"log"`
	APIs APIsConfig `yaml:"apis" json:"apis"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`
}

type GRPCConfig struct {
	ReflectedServices []string `yaml:"reflectedServices" json:"reflectedServices"`
	server.BaseConfig `yaml:",inline" json:",inline"`
}

var allowedReflectedServices = map[string]struct{}{
	grpcGatewayPb.GrpcGateway_ServiceDesc.ServiceName:            {},
	certAuthorityPb.CertificateAuthority_ServiceDesc.ServiceName: {},
	snippetServicePb.SnippetService_ServiceDesc.ServiceName:      {},
	m2mOAuthServerPb.M2MOAuthService_ServiceDesc.ServiceName:     {},
}

func (c *GRPCConfig) Validate() error {
	// Check if ReflectedServices is not empty
	if len(c.ReflectedServices) == 0 {
		return errors.New("reflectedServices[] - is empty")
	}

	// Check if each service name is not empty
	for idx, service := range c.ReflectedServices {
		_, ok := allowedReflectedServices[service]
		if !ok {
			return fmt.Errorf("reflectedServices[%v]('%v') - is invalid", idx, service)
		}
	}

	// Validate the embedded server.Config
	if err := c.BaseConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
