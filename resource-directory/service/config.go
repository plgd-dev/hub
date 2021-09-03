package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/cloud/coap-gateway/authorization"
	"github.com/plgd-dev/cloud/grpc-gateway/pb"
	"github.com/plgd-dev/cloud/pkg/config"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/pkg/net/grpc/client"
	"github.com/plgd-dev/cloud/pkg/net/grpc/server"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	natsClient "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/client"
	eventstoreConfig "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/config"
)

type Config struct {
	Log                       log.Config          `yaml:"log" json:"log"`
	APIs                      APIsConfig          `yaml:"apis" json:"apis"`
	Clients                   ClientsConfig       `yaml:"clients" json:"clients"`
	ExposedCloudConfiguration PublicConfiguration `yaml:"publicConfiguration" json:"publicConfiguration"`
}

func (c *Config) Validate() error {
	err := c.APIs.Validate()
	if err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	err = c.Clients.Validate()
	if err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	err = c.ExposedCloudConfiguration.Validate()
	if err != nil {
		return fmt.Errorf("publicConfiguration.%w", err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC server.Config `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	err := c.GRPC.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ClientsConfig struct {
	Eventbus   EventBusConfig            `yaml:"eventBus" json:"eventBus"`
	Eventstore EventStoreConfig          `yaml:"eventStore" json:"eventStore"`
	AuthServer AuthorizationServerConfig `yaml:"authorizationServer" json:"authorizationServer"`
}

type EventBusConfig struct {
	GoPoolSize int               `yaml:"goPoolSize" json:"goPoolSize"`
	NATS       natsClient.Config `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	if c.GoPoolSize <= 0 {
		return fmt.Errorf("goPoolSize('%v')", c.GoPoolSize)
	}
	err := c.NATS.Validate()
	if err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
}

type EventStoreConfig struct {
	ProjectionCacheExpiration time.Duration           `yaml:"cacheExpiration" json:"cacheExpiration"`
	Connection                eventstoreConfig.Config `yaml:",inline" json:",inline"`
}

func (c *EventStoreConfig) Validate() error {
	if c.ProjectionCacheExpiration <= 0 {
		return fmt.Errorf("cacheExpiration('%v')", c.ProjectionCacheExpiration)
	}
	return c.Connection.Validate()
}

func (c *ClientsConfig) Validate() error {
	err := c.AuthServer.Validate()
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

type AuthorizationServerConfig struct {
	PullFrequency   time.Duration    `yaml:"pullFrequency" json:"pullFrequency"`
	CacheExpiration time.Duration    `yaml:"cacheExpiration" json:"cacheExpiration"`
	OwnerClaim      string           `yaml:"ownerClaim" json:"ownerClaim"`
	Connection      client.Config    `yaml:"grpc" json:"grpc"`
	OAuth           manager.ConfigV2 `yaml:"oauth" json:"oauth"`
}

func (c *AuthorizationServerConfig) Validate() error {
	if c.PullFrequency <= 0 {
		return fmt.Errorf("pullFrequency('%v')", c.PullFrequency)
	}
	if c.CacheExpiration <= 0 {
		return fmt.Errorf("cacheExpiration('%v')", c.CacheExpiration)
	}
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	err := c.OAuth.Validate()
	if err != nil {
		return fmt.Errorf("oauth.%w", err)
	}
	err = c.Connection.Validate()
	if err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return err
}

type PublicConfiguration struct {
	CAPool                     string               `yaml:"caPool" json:"caPool" description:"file path to the root certificate in PEM format"`
	DeviceAuthorization        authorization.Config `yaml:"deviceAuthorization" json:"deviceAuthorization"`
	OwnerClaim                 string               `yaml:"ownerClaim" json:"ownerClaim"`
	SigningServerAddress       string               `yaml:"signingServerAddress" json:"signingServerAddress"`
	CloudID                    string               `yaml:"cloudID" json:"cloudID"`
	CloudURL                   string               `yaml:"cloudURL" json:"cloudURL"`
	CloudAuthorizationProvider string               `yaml:"cloudAuthorizationProvider" json:"cloudAuthorizationProvider"`
	DefaultCommandTimeToLive   time.Duration        `yaml:"defaultCommandTimeToLive" json:"defaultCommandTimeToLive"`

	cloudCertificateAuthorities string `yaml:"-"`
}

func (c *PublicConfiguration) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	if c.CloudID == "" {
		return fmt.Errorf("cloudID('%v')", c.CloudID)
	}
	if c.CloudURL == "" {
		return fmt.Errorf("cloudURL('%v')", c.CloudURL)
	}
	if c.CloudAuthorizationProvider == "" {
		return fmt.Errorf("cloudAuthorizationProvider('%v')", c.CloudAuthorizationProvider)
	}
	if c.CAPool == "" {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	if err := c.DeviceAuthorization.Validate(); err != nil {
		return fmt.Errorf("deviceAuthorization.%w", err)
	}
	return nil
}

func (c PublicConfiguration) ToProto(authURL string) *pb.CloudConfigurationResponse {
	return &pb.CloudConfigurationResponse{
		DeviceOnboardingCodeUrl:     c.DeviceAuthorization.AuthCodeURL(uuid.NewString(), authURL, ""),
		JwtClaimOwnerId:             c.OwnerClaim,
		SigningServerAddress:        c.SigningServerAddress,
		CloudId:                     c.CloudID,
		CloudUrl:                    c.CloudURL,
		CloudAuthorizationProvider:  c.CloudAuthorizationProvider,
		CloudCertificateAuthorities: c.cloudCertificateAuthorities,
		DefaultCommandTimeToLive:    int64(c.DefaultCommandTimeToLive),
		CurrentTime:                 pkgTime.UnixNano(time.Now()),
	}
}

//String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
