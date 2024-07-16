package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/build"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	eventstoreConfig "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/config"
)

type Config struct {
	HubID                   string              `yaml:"hubID" json:"hubId"`
	Log                     log.Config          `yaml:"log" json:"log"`
	APIs                    APIsConfig          `yaml:"apis" json:"apis"`
	Clients                 ClientsConfig       `yaml:"clients" json:"clients"`
	ExposedHubConfiguration PublicConfiguration `yaml:"publicConfiguration" json:"publicConfiguration"`
}

func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if err := c.ExposedHubConfiguration.Validate(); err != nil {
		return fmt.Errorf("publicConfiguration.%w", err)
	}
	if _, err := uuid.Parse(c.HubID); err != nil {
		return fmt.Errorf("hubID('%v') - %w", c.HubID, err)
	}
	return nil
}

// Config represent application configuration
type APIsConfig struct {
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`
}

func (c *APIsConfig) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type GRPCConfig struct {
	OwnerCacheExpiration time.Duration `yaml:"ownerCacheExpiration" json:"ownerCacheExpiration"`
	server.Config        `yaml:",inline" json:",inline"`
}

func (c *GRPCConfig) Validate() error {
	if c.OwnerCacheExpiration <= 0 {
		return fmt.Errorf("ownerCacheExpiration('%v')", c.OwnerCacheExpiration)
	}
	return c.Config.Validate()
}

type ClientsConfig struct {
	Eventbus               EventBusConfig      `yaml:"eventBus" json:"eventBus"`
	Eventstore             EventStoreConfig    `yaml:"eventStore" json:"eventStore"`
	IdentityStore          IdentityStoreConfig `yaml:"identityStore" json:"identityStore"`
	OpenTelemetryCollector otelClient.Config   `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.IdentityStore.Validate(); err != nil {
		return fmt.Errorf("identityStore.%w", err)
	}
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	if err := c.Eventstore.Validate(); err != nil {
		return fmt.Errorf("eventstore.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

type EventBusConfig struct {
	GoPoolSize int                         `yaml:"goPoolSize" json:"goPoolSize"`
	NATS       natsClient.ConfigSubscriber `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	if c.GoPoolSize <= 0 {
		return fmt.Errorf("goPoolSize('%v')", c.GoPoolSize)
	}
	if err := c.NATS.Validate(); err != nil {
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

type IdentityStoreConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *IdentityStoreConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type PublicConfiguration struct {
	CAPool        urischeme.URIScheme `yaml:"caPool" json:"caPool" description:"file path to the root certificate in PEM format"`
	OwnerClaim    string              `yaml:"ownerClaim" json:"ownerClaim"`
	DeviceIDClaim string              `yaml:"deviceIDClaim" json:"deviceIdClaim"`

	CoapGateway              string        `yaml:"coapGateway" json:"coapGateway"`
	DefaultCommandTimeToLive time.Duration `yaml:"defaultCommandTimeToLive" json:"defaultCommandTimeToLive"`
	Authority                string        `yaml:"authority" json:"authority"`
	CertificateAuthority     string        `yaml:"certificateAuthority" json:"certificateAuthority"`

	cloudCertificateAuthorities string `yaml:"-"`
}

func (c *PublicConfiguration) Validate() error {
	if c.CAPool == "" {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	if c.CoapGateway == "" {
		return fmt.Errorf("coapGateway('%v')", c.CoapGateway)
	}
	if c.CAPool == "" {
		return fmt.Errorf("caPool('%v')", c.CAPool)
	}
	if c.Authority == "" {
		return fmt.Errorf("authority('%v')", c.Authority)
	}
	if c.CertificateAuthority == "" {
		return fmt.Errorf("certificateAuthority('%v')", c.CertificateAuthority)
	}
	return nil
}

func (c PublicConfiguration) ToProto(hubID string) *pb.HubConfigurationResponse {
	return &pb.HubConfigurationResponse{
		JwtOwnerClaim:            c.OwnerClaim,
		JwtDeviceIdClaim:         c.DeviceIDClaim,
		Id:                       hubID,
		CoapGateway:              c.CoapGateway,
		CertificateAuthorities:   c.cloudCertificateAuthorities,
		DefaultCommandTimeToLive: int64(c.DefaultCommandTimeToLive),
		CurrentTime:              pkgTime.UnixNano(time.Now()),
		Authority:                c.Authority,
		CertificateAuthority:     c.CertificateAuthority,
		BuildInfo: &pb.BuildInfo{
			Version:    build.Version,
			CommitHash: build.CommitHash,
			BuildDate:  build.BuildDate,
			CommitDate: build.CommitDate,
			ReleaseUrl: build.ReleaseURL,
		},
	}
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
