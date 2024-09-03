package service

import (
	"fmt"
	"hash/crc64"
	"time"

	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/security/oauth/clientcredentials"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/http"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/internal/math"
	"github.com/plgd-dev/hub/v2/pkg/config"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgCoapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	pkgHttpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	pkgCertManagerClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/strings"
)

// Config represents application configuration
type Config struct {
	Log              LogConfig        `yaml:"log" json:"log"`
	APIs             APIsConfig       `yaml:"apis" json:"apis"`
	Clients          ClientsConfig    `yaml:"clients" json:"clients"`
	EnrollmentGroups EnrollmentGroups `yaml:"enrollmentGroups" json:"enrollmentGroups"`
}

func (c *Config) Validate() error {
	if err := c.APIs.Validate(); err != nil {
		return fmt.Errorf("apis.%w", err)
	}
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log.%w", err)
	}
	if err := c.Clients.Validate(); err != nil {
		return fmt.Errorf("clients.%w", err)
	}
	if len(c.EnrollmentGroups) == 0 {
		// EnrollmentGroups are optional because they can be added later through the HTTP API
		return nil
	}
	for i := range c.EnrollmentGroups {
		if err := c.EnrollmentGroups[i].Validate(); err != nil {
			return fmt.Errorf("enrollmentGroups[%v].%w", i, err)
		}
	}
	return nil
}

// LogConfig represents application configuration
type LogConfig = log.Config

type GrpcClientConfig struct {
	Connection client.Config `yaml:"grpc" json:"grpc"`
}

func (c *GrpcClientConfig) Validate() error {
	if err := c.Connection.Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

func TLSConfigToProto(cfg pkgCertManagerClient.Config) (*pb.TlsConfig, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	caPool, err := cfg.CAPoolArray()
	if err != nil {
		return nil, err
	}
	caPoolStr := make([]string, 0, len(caPool))
	for _, ca := range caPool {
		caPoolStr = append(caPoolStr, string(ca))
	}

	return &pb.TlsConfig{
		CaPool:          caPoolStr,
		Cert:            string(cfg.CertFile),
		Key:             string(cfg.KeyFile),
		UseSystemCaPool: cfg.UseSystemCAPool,
	}, nil
}

func (c *GrpcClientConfig) ToProto() (*pb.GrpcClientConfig, error) {
	tls, err := TLSConfigToProto(c.Connection.TLS)
	if err != nil {
		return nil, err
	}

	return &pb.GrpcClientConfig{
		Grpc: &pb.GrpcConnectionConfig{
			Address: c.Connection.Addr,
			KeepAlive: &pb.GrpcKeepAliveConfig{
				PermitWithoutStream: c.Connection.KeepAlive.PermitWithoutStream,
				Time:                c.Connection.KeepAlive.Time.Nanoseconds(),
				Timeout:             c.Connection.KeepAlive.Timeout.Nanoseconds(),
			},
			Tls: tls,
		},
	}, nil
}

type HTTPConfig struct {
	http.Config `yaml:",inline"`
	Enabled     bool `yaml:"enabled" json:"enabled"`
}

func (c *HTTPConfig) Validate() error {
	return c.Config.Validate()
}

type APIsConfig struct {
	COAP COAPConfig `yaml:"coap" json:"coap"`
	HTTP HTTPConfig `yaml:"http" json:"http"`
}

func (c *APIsConfig) Validate() error {
	if err := c.COAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	if c.HTTP.Enabled {
		if err := c.HTTP.Validate(); err != nil {
			return fmt.Errorf("http.%w", err)
		}
	}
	return nil
}

type COAPConfig struct {
	pkgCoapService.Config `yaml:",inline" json:",inline"`
}

func (c *COAPConfig) Validate() error {
	c.TLS.Embedded.CAPoolIsOptional = true
	c.TLS.Embedded.ClientCertificateRequired = true
	enabled := true
	c.TLS.Enabled = &enabled
	return c.Config.Validate()
}

// String returns string representation of Config
func (c Config) String() string {
	return config.ToString(c)
}

type EnrollmentGroups []EnrollmentGroupConfig

func (g EnrollmentGroups) FindByID(id string) (EnrollmentGroupConfig, bool) {
	for _, eg := range g {
		if eg.ID == id {
			return eg, true
		}
	}
	return EnrollmentGroupConfig{}, false
}

func toCRC64(d []byte) uint64 {
	c := crc64.New(crc64.MakeTable(crc64.ISO))
	c.Write(d)
	return c.Sum64()
}

type X509 struct {
	CertificateChain          urischeme.URIScheme `yaml:"certificateChain" json:"certificateChain"`
	LeadCertificateName       string              `yaml:"leadCertificateName" json:"leadCertificateName"`
	ExpiredCertificateEnabled bool                `yaml:"expiredCertificateEnabled" json:"expiredCertificateEnabled"`
}

func (c *X509) ToProto() *pb.X509Configuration {
	return &pb.X509Configuration{
		CertificateChain:          string(c.CertificateChain),
		ExpiredCertificateEnabled: c.ExpiredCertificateEnabled,
		LeadCertificateName:       c.LeadCertificateName,
	}
}

type AttestationMechanism struct {
	X509 X509 `yaml:"x509" json:"x509"`
}

func (c *AttestationMechanism) ToProto() *pb.AttestationMechanism {
	return &pb.AttestationMechanism{
		X509: c.X509.ToProto(),
	}
}

type EnrollmentGroupConfig struct {
	ID                   string               `yaml:"id" json:"id"`
	Owner                string               `yaml:"owner" json:"owner"`
	AttestationMechanism AttestationMechanism `yaml:"attestationMechanism" json:"attestationMechanism"`
	Hub                  HubConfig            `yaml:"hub" json:"hub"`
	Hubs                 []HubConfig          `yaml:"hubs" json:"hubs"`
	PreSharedKeyFile     urischeme.URIScheme  `yaml:"preSharedKeyFile" json:"preSharedKeyFile"`
	Name                 string               `yaml:"name" json:"name"`
}

func (e *EnrollmentGroupConfig) ToProto() (*pb.EnrollmentGroup, []*pb.Hub, error) {
	attestationMechanism := e.AttestationMechanism.ToProto()
	hubIDs := make([]string, 0, len(e.Hubs))
	hubs := make([]*pb.Hub, 0, len(e.Hubs))
	if e.Hub.ID != "" || e.Hub.HubID != "" {
		hub, err := e.Hub.ToProto(e.Owner)
		if err == nil {
			err = hub.Validate(e.Owner)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("hub.%w", err)
		}
		hubIDs = append(hubIDs, hub.GetHubId())
		hubs = append(hubs, hub)
	}
	for idx, hub := range e.Hubs {
		hub, err := hub.ToProto(e.Owner)
		if err == nil {
			err = hub.Validate(e.Owner)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("hubs[%v].%w", idx, err)
		}
		hubIDs = append(hubIDs, hub.GetHubId())
		hubs = append(hubs, hub)
	}
	eg := &pb.EnrollmentGroup{
		Id:                   e.ID,
		Owner:                e.Owner,
		AttestationMechanism: attestationMechanism,
		HubIds:               hubIDs,
		PreSharedKey:         string(e.PreSharedKeyFile),
		Name:                 e.Name,
	}
	if err := eg.Validate(e.Owner); err != nil {
		return nil, nil, err
	}
	return eg, hubs, nil
}

func (e *EnrollmentGroupConfig) String() string {
	return e.ID
}

func (e *EnrollmentGroupConfig) Validate() error {
	_, _, err := e.ToProto()
	return err
}

type AuthorizationProviderConfig struct {
	Name                     string `yaml:"name" json:"name"`
	clientcredentials.Config `yaml:",inline"`
}

func HTTPConfigToProto(cfg pkgHttpClient.Config) (*pb.HttpConfig, error) {
	tls, err := TLSConfigToProto(cfg.TLS)
	if err != nil {
		return nil, err
	}

	return &pb.HttpConfig{
		MaxIdleConns:        math.CastTo[uint32](cfg.MaxIdleConns),
		MaxConnsPerHost:     math.CastTo[uint32](cfg.MaxConnsPerHost),
		MaxIdleConnsPerHost: math.CastTo[uint32](cfg.MaxIdleConnsPerHost),
		IdleConnTimeout:     cfg.IdleConnTimeout.Nanoseconds(),
		Timeout:             cfg.Timeout.Nanoseconds(),
		Tls:                 tls,
	}, nil
}

func (c *AuthorizationProviderConfig) ToProto() (*pb.AuthorizationProviderConfig, error) {
	http, err := HTTPConfigToProto(c.HTTP)
	if err != nil {
		return nil, err
	}

	return &pb.AuthorizationProviderConfig{
		Name:         c.Name,
		Authority:    c.Authority,
		ClientId:     c.ClientID,
		ClientSecret: string(c.ClientSecretFile),
		Scopes:       c.Scopes,
		Audience:     c.Audience,
		Http:         http,
	}, nil
}

type AuthorizationConfig struct {
	OwnerClaim    string                      `yaml:"ownerClaim" json:"ownerClaim"`
	DeviceIDClaim string                      `yaml:"deviceIDClaim" json:"deviceIdClaim"`
	Provider      AuthorizationProviderConfig `yaml:"provider" json:"provider"`
}

func (c *AuthorizationConfig) ToProto() (*pb.AuthorizationConfig, error) {
	provider, err := c.Provider.ToProto()
	if err != nil {
		return nil, fmt.Errorf("provider.%w", err)
	}
	return &pb.AuthorizationConfig{
		Provider:      provider,
		OwnerClaim:    c.OwnerClaim,
		DeviceIdClaim: c.DeviceIDClaim,
	}, nil
}

type HubConfig struct {
	ID                   string              `yaml:"id" json:"id"`
	HubID                string              `yaml:"hubID" json:"hubId"`
	CoapGateway          string              `yaml:"coapGateway" json:"coapGateway"`
	Gateways             []string            `yaml:"gateways" json:"gateways"`
	CertificateAuthority GrpcClientConfig    `yaml:"certificateAuthority" json:"certificateAuthority"`
	Authorization        AuthorizationConfig `yaml:"authorization" json:"authorization"`
	Name                 string              `yaml:"name" json:"name"`
}

func (c *HubConfig) Validate(owner string) error {
	p, err := c.ToProto(owner)
	if err != nil {
		return err
	}
	return p.Validate(owner)
}

func (c *HubConfig) ToProto(owner string) (*pb.Hub, error) {
	authorization, err := c.Authorization.ToProto()
	if err != nil {
		return nil, fmt.Errorf("authorization.%w", err)
	}
	certificateAuthority, err := c.CertificateAuthority.ToProto()
	if err != nil {
		return nil, fmt.Errorf("authorization.%w", err)
	}
	coapGWs := make([]string, 0, len(c.Gateways)+1)
	if c.CoapGateway != "" {
		coapGWs = append(coapGWs, c.CoapGateway)
	}
	coapGWs = append(coapGWs, c.Gateways...)
	if c.ID == "" {
		c.ID = c.HubID
	}
	if c.HubID == "" {
		c.HubID = c.ID
	}

	return &pb.Hub{
		Id:                   c.ID,
		HubId:                c.HubID,
		Gateways:             strings.UniqueStable(coapGWs),
		CertificateAuthority: certificateAuthority,
		Authorization:        authorization,
		Name:                 c.Name,
		Owner:                owner,
	}, nil
}

type CoapConfig struct {
	Address string `yaml:"address" json:"address"`
}

func (c *CoapConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("address('%v')", c.Address)
	}
	return nil
}

type CoapGatewayConfig struct {
	COAP         CoapConfig `yaml:"coap" json:"coap"`
	ProviderName string     `yaml:"providerName" json:"providerName"`
}

func (c *CoapGatewayConfig) Validate() error {
	if err := c.COAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	if c.ProviderName == "" {
		return fmt.Errorf("providerName('%v')", c.ProviderName)
	}
	return nil
}

type ClientsConfig struct {
	Storage                StorageConfig                        `yaml:"storage" json:"storage"`
	OpenTelemetryCollector pkgHttp.OpenTelemetryCollectorConfig `yaml:"openTelemetryCollector" json:"openTelemetryCollector"`
}

func (c *ClientsConfig) Validate() error {
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage.%w", err)
	}
	if err := c.OpenTelemetryCollector.Validate(); err != nil {
		return fmt.Errorf("openTelemetryCollector.%w", err)
	}
	return nil
}

type StorageConfig struct {
	// expiration time of cached DB records
	CacheExpiration time.Duration  `yaml:"cacheExpiration" json:"cacheExpiration"`
	MongoDB         mongodb.Config `yaml:"mongoDB" json:"mongoDb"` //nolint:tagliatelle
}

func (c *StorageConfig) Validate() error {
	if c.CacheExpiration < time.Second {
		return fmt.Errorf("cacheExpiration('%v') - less than %v", c.CacheExpiration, time.Second)
	}
	if err := c.MongoDB.Validate(); err != nil {
		return fmt.Errorf("mongoDB.%w", err)
	}
	return nil
}
