package service

import (
	"fmt"
	"time"

	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	eventbusConfig "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/config"
	eventstoreConfig "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/config"
)

type Config struct {
	Log     log.Config    `yaml:"log" json:"log" envconfig:"LOG"`
	APIs    APIsConfig    `yaml:"apis" json:"apis" envconfig:"SERVICE"`
	Clients ClientsConfig `yaml:"clients" json:"clients" envconfig:"CLIENTS"`
}

// Config represent application configuration
type APIsConfig struct {
	GRPC GrpcConfig `yaml:"grpc" json:"grpc" envconfig:"GRPC"`
}

type GrpcConfig struct {
	Server       server.Config      `yaml:",inline" json:",inline"`
	Capabilities CapabilitiesConfig `yaml:"capabilities" json:"capabilities"`
}

type CapabilitiesConfig struct {
	TimeoutForRequests              time.Duration `yaml:"timeout" json:"timeout" envconfig:"TIMEOUT" default:"10s"`
	ProjectionCacheExpiration       time.Duration `yaml:"cacheExpiration" json:"cacheExpiration" envconfig:"CACHE_EXPIRATION" default:"1m"`
	GoRoutinePoolSize               int           `yaml:"goRoutinePoolSize" json:"goRoutinePoolSize" envconfig:"GOROUTINE_POOL_SIZE" default:"16"`
	UserDevicesManagerTickFrequency time.Duration `yaml:"userMgmtTickFrequency" json:"userMgmtTickFrequency" envconfig:"USER_MGMT_TICK_FREQUENCY" default:"15s"`
	UserDevicesManagerExpiration    time.Duration `yaml:"userMgmtExpiration" json:"userMgmtExpiration" envconfig:"USER_MGMT_EXPIRATION" default:"1m"`
}

type ClientsConfig struct {
	Eventbus            eventbusConfig.Config   `yaml:"eventBus" json:"eventBus"`
	Eventstore          eventstoreConfig.Config `yaml:"eventStore" json:"eventStore"`
	OAuthProvider       OAuthProviderConfig     `yaml:"oauthProvider" json:"oauthProvider"`
	AuthServer          client.Config           `yaml:"authorizationServer" json:"authorizationServer"`
	ClientConfiguration CloudConfig             `yaml:"clientConfiguration" envconfig:"CONFIG"`
}

func (c *ClientsConfig) Validate() error {
	err := c.OAuthProvider.Validate()
	if err != nil {
		return fmt.Errorf("oauthProvider.%w", err)
	}
	err = c.AuthServer.Validate()
	if err != nil {
		return fmt.Errorf("authorizationServer.%w", err)
	}
	err = c.Eventbus.Validate()
	if err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	err = c.Eventstore.Validate()
	if err != nil {
		return fmt.Errorf("eventstore.%w", err)
	}
	return nil
}

type OAuthProviderConfig struct {
	Jwks       validator.Config `yaml:"jwks" json:"jwks"`
	OAuth      manager.ConfigV2 `yaml:"oauth" json:"oauth"`
	OwnerClaim string           `yaml:"ownerClaim" json:"ownerClaim"`
}

func (c *OAuthProviderConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	err := c.Jwks.Validate()
	if err != nil {
		return fmt.Errorf("jwks.%w", err)
	}
	err = c.OAuth.Validate()
	if err != nil {
		return fmt.Errorf("oauth.%w", err)
	}
	return nil
}

type CloudConfig struct {
	CloudCAPool                    string `yaml:"cloudCAPool" json:"cloudCAPool" envconfig:"CLOUD_CA_POOL" description:"file path to the root certificate in PEM format"`
	pb.ClientConfigurationResponse `yaml:",inline"`
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
