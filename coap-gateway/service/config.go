package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	"github.com/plgd-dev/hub/v2/pkg/sync/task/queue"
	pkgYaml "github.com/plgd-dev/hub/v2/pkg/yaml"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"gopkg.in/yaml.v3"
)

// Config represent application configuration
type Config struct {
	Log              LogConfig              `yaml:"log" json:"log"`
	APIs             APIsConfig             `yaml:"apis" json:"apis"`
	Clients          ClientsConfig          `yaml:"clients" json:"clients"`
	DeviceTwin       DeviceTwinConfig       `yaml:"deviceTwin" json:"deviceTwin"`
	TaskQueue        queue.Config           `yaml:"taskQueue" json:"taskQueue"`
	ServiceHeartbeat ServiceHeartbeatConfig `yaml:"serviceHeartbeat" json:"serviceHeartbeat"`
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
	if err := c.TaskQueue.Validate(); err != nil {
		return fmt.Errorf("taskQueue.%w", err)
	}
	if err := c.ServiceHeartbeat.Validate(); err != nil {
		return fmt.Errorf("serviceHeartbeat.%w", err)
	}
	return nil
}

type ServiceHeartbeatConfig struct {
	TimeToLive time.Duration `yaml:"timeToLive" json:"timeToLive"`
}

func (c *ServiceHeartbeatConfig) Validate() error {
	minTimeToLive := time.Second
	if c.TimeToLive < minTimeToLive {
		return fmt.Errorf("timeToLive('%v') - is less than %v", c.TimeToLive, minTimeToLive)
	}
	return nil
}

type LogConfig = log.Config

type APIsConfig struct {
	COAP COAPConfigMarshalerUnmarshaler `yaml:"coap" json:"coap"`
}

func (c *APIsConfig) Validate() error {
	if err := c.COAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	return nil
}

type ProvidersConfig struct {
	Name          string `yaml:"name" json:"name"`
	oauth2.Config `yaml:",inline"`
}

func (c *ProvidersConfig) Validate(firstAuthority string, providerNames map[string]bool) error {
	if c.Authority != firstAuthority {
		return fmt.Errorf("authority('%v' != authorization[0].authority('%v'))", c.Authority, firstAuthority)
	}
	if _, ok := providerNames[c.Name]; ok {
		return fmt.Errorf("name('%v' is duplicit)", c.Name)
	}
	providerNames[c.Name] = true
	return c.Config.Validate()
}

type AuthorizationConfig struct {
	DeviceIDClaim string            `yaml:"deviceIDClaim" json:"deviceIdClaim"`
	OwnerClaim    string            `yaml:"ownerClaim" json:"ownerClaim"`
	Providers     []ProvidersConfig `yaml:"providers" json:"providers"`
	Authority     validator.Config  `yaml:",inline" json:",inline"`
}

func (c *AuthorizationConfig) Validate() error {
	if c.OwnerClaim == "" {
		return fmt.Errorf("ownerClaim('%v')", c.OwnerClaim)
	}
	if len(c.Providers) == 0 {
		return fmt.Errorf("providers('%v')", c.Providers)
	}
	duplicitProviderNames := make(map[string]bool, 4)
	for i := range len(c.Providers) {
		if c.Providers[i].GrantType == oauth.ClientCredentials && c.OwnerClaim == "sub" {
			return fmt.Errorf("providers[%v].grantType - %w", i, fmt.Errorf("combination of ownerClaim set to '%v' is not compatible if at least one authorization provider uses grant type '%v'", c.OwnerClaim, c.Providers[i].GrantType))
		}
		if err := c.Providers[i].Validate(c.Providers[0].Authority, duplicitProviderNames); err != nil {
			return fmt.Errorf("providers[%v].%w", i, err)
		}
	}
	// for backward compatibility
	if c.Authority.Authority == nil {
		c.Authority.Authority = &c.Providers[0].Authority
	}
	if c.Authority.HTTP == nil {
		c.Authority.HTTP = &c.Providers[0].HTTP
	}
	if err := c.Authority.Validate(); err != nil {
		return fmt.Errorf("authority.%w", err)
	}
	return nil
}

type InjectedTLSConfig struct {
	IdentityPropertiesRequired bool `yaml:"identityPropertiesRequired" json:"identityPropertiesRequired"`
}

type InjectedCOAPConfig struct {
	TLSConfig InjectedTLSConfig `yaml:"tls" json:"tls"`
}

type DeviceTwinConfig struct {
	MaxETagsCountInRequest uint32 `yaml:"maxETagsCountInRequest" json:"maxETagsCountInRequest"`
	UseETags               bool   `yaml:"useETags" json:"useETags"`
}

type COAPConfig struct {
	coapService.Config         `yaml:",inline" json:",inline"`
	ExternalAddress            string              `yaml:"externalAddress" json:"externalAddress"`
	Authorization              AuthorizationConfig `yaml:"authorization" json:"authorization"`
	OwnerCacheExpiration       time.Duration       `yaml:"ownerCacheExpiration" json:"ownerCacheExpiration"`
	SubscriptionBufferSize     int                 `yaml:"subscriptionBufferSize" json:"subscriptionBufferSize"`
	RequireBatchObserveEnabled bool                `yaml:"requireBatchObserveEnabled" json:"requireBatchObserveEnabled"`

	InjectedCOAPConfig InjectedCOAPConfig `yaml:"-" json:"-"`
}

type COAPConfigMarshalerUnmarshaler struct {
	COAPConfig `yaml:",inline"`
}

func (c *COAPConfigMarshalerUnmarshaler) UnmarshalYAML(value *yaml.Node) error {
	err := value.Decode(&c.COAPConfig)
	if err != nil {
		return err
	}
	return value.Decode(&c.InjectedCOAPConfig)
}

func (c COAPConfigMarshalerUnmarshaler) MarshalYAML() (interface{}, error) {
	node := yaml.Node{}
	err := node.Encode(c.COAPConfig)
	if err != nil {
		return nil, err
	}
	injectNode := yaml.Node{}
	err = injectNode.Encode(c.InjectedCOAPConfig)
	if err != nil {
		return nil, err
	}
	return pkgYaml.MergeYamlNodes(&node, &injectNode)
}

func (c *COAPConfigMarshalerUnmarshaler) Validate() error {
	if err := c.COAPConfig.Validate(); err != nil {
		return err
	}
	if !c.InjectedCOAPConfig.TLSConfig.IdentityPropertiesRequired && c.Authorization.DeviceIDClaim != "" {
		return fmt.Errorf("tls.identityPropertiesRequired('%v') - %w", c.InjectedCOAPConfig.TLSConfig.IdentityPropertiesRequired, errors.New("combination with authorization.deviceIDClaim is not supported"))
	}
	return nil
}

func (c *COAPConfig) Validate() error {
	if c.ExternalAddress == "" {
		return fmt.Errorf("externalAddress('%v')", c.ExternalAddress)
	}
	if c.OwnerCacheExpiration <= 0 {
		return fmt.Errorf("ownerCacheExpiration('%v')", c.OwnerCacheExpiration)
	}
	if c.SubscriptionBufferSize < 0 {
		return fmt.Errorf("subscriptionBufferSize('%v')", c.SubscriptionBufferSize)
	}
	if err := c.Authorization.Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	return c.Config.Validate()
}

type EventBusConfig struct {
	NATS natsClient.ConfigSubscriber `yaml:"nats" json:"nats"`
}

func (c *EventBusConfig) Validate() error {
	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats.%w", err)
	}
	return nil
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

type ClientsConfig struct {
	Eventbus               EventBusConfig          `yaml:"eventBus" json:"eventBus"`
	OpenTelemetryCollector otelClient.Config       `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
	IdentityStore          IdentityStoreConfig     `yaml:"identityStore" json:"identityStore"`
	ResourceDirectory      GrpcServerConfig        `yaml:"resourceDirectory" json:"resourceDirectory"`
	CertificateAuthority   GrpcServerConfig        `yaml:"certificateAuthority" json:"certificateAuthority"`
	ResourceAggregate      ResourceAggregateConfig `yaml:"resourceAggregate" json:"resourceAggregate"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.IdentityStore.Validate(); err != nil {
		return fmt.Errorf("identityStore.%w", err)
	}
	if err := c.Eventbus.Validate(); err != nil {
		return fmt.Errorf("eventbus.%w", err)
	}
	if err := c.ResourceAggregate.Validate(); err != nil {
		return fmt.Errorf("resourceAggregate.%w", err)
	}
	if err := c.ResourceDirectory.Validate(); err != nil {
		return fmt.Errorf("resourceDirectory.%w", err)
	}
	if err := c.CertificateAuthority.Validate(); err != nil {
		return fmt.Errorf("certificateAuthority.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

type GrpcServerConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *GrpcServerConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

type ResourceAggregateConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *ResourceAggregateConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

// String return string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}
