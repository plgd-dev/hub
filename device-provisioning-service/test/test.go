package test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/collection"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/pb"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/service/http"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/store"
	storeMongo "github.com/plgd-dev/hub/v2/device-provisioning-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/identity-store/events"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	pkgCoapService "github.com/plgd-dev/hub/v2/pkg/net/coap/service"
	pkgHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/sdk"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	DPSHost              = "127.0.0.1:20130"
	DPSHTTPHost          = "127.0.0.1:20131"
	DPSEnrollmentGroupID = "6aa1aa8e-2b91-48ee-bfbc-a4e22d8e20d8"
	DPSOwner             = "1"
)

var (
	TestDockerContainerName             string
	TestDeviceName                      string
	TestDockerObtContainerName          string
	TestDeviceObtName                   string
	TestDeviceObtSupportsTestProperties bool

	TestDevsimResources []schema.ResourceLink
)

const (
	TestResourceSwitchesHref = "/switches"
	ResourcePlgdDpsHref      = "/plgd/dps"
	ResourcePlgdDpsType      = "x.plgd.dps.conf"
)

type ResourcePlgdDpsTestCloudStatusObserver struct {
	MaxCount int64 `json:"maxCount,omitempty"`
}

type ResourcePlgdDpsTestIotivity struct {
	Retry []int64 `json:"retry,omitempty"`
}

type ResourcePlgdDpsTest struct {
	CloudStatusObserver ResourcePlgdDpsTestCloudStatusObserver `json:"cloudStatusObserver,omitempty"`
	Iotivity            ResourcePlgdDpsTestIotivity            `json:"iotivity,omitempty"`
}

type DpsEndpoint struct {
	URI  string `json:"uri,omitempty"`
	Name string `json:"name,omitempty"`
}

type ResourcePlgdDps struct {
	Endpoint         *string             `json:"endpoint,omitempty"`
	EndpointName     *string             `json:"endpointName,omitempty"`
	Endpoints        []DpsEndpoint       `json:"endpoints,omitempty"`
	LastError        uint64              `json:"lastErrorCode,omitempty"`
	ProvisionStatus  string              `json:"provisionStatus,omitempty"`
	ForceReprovision bool                `json:"forceReprovision,omitempty"`
	Interfaces       []string            `json:"if,omitempty"`
	ResourceTypes    []string            `json:"rt,omitempty"`
	TestProperties   ResourcePlgdDpsTest `json:"test,omitempty"`
}

func CleanUpDpsResource(dps *ResourcePlgdDps) {
	dps.TestProperties = ResourcePlgdDpsTest{}
}

func LightResourceInstanceHref(id string) string {
	return "/light/" + id
}

func checkDpsResourceForTestProperties() bool {
	deviceID := hubTest.MustFindDeviceByName(TestDeviceObtName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	devClient, err := sdk.NewClient(sdk.WithID(events.OwnerToUUID(DPSOwner)))
	if err != nil {
		return false
	}
	defer func() {
		_ = devClient.Close(ctx)
	}()
	deviceID, err = devClient.OwnDevice(ctx, deviceID, deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	if err != nil {
		return false
	}
	defer func() {
		_ = devClient.DisownDevice(ctx, deviceID)
	}()
	var resp map[string]interface{}
	err = devClient.GetResource(ctx, deviceID, ResourcePlgdDpsHref, &resp)
	if err != nil {
		return false
	}

	_, ok := resp["test"]
	return ok
}

func init() {
	TestDockerContainerName = "dps-devsim"
	TestDeviceName = TestDockerContainerName + "-" + hubTest.MustGetHostname()
	TestDockerObtContainerName = "dps-devsim-obt"
	TestDeviceObtName = TestDockerObtContainerName + "-" + hubTest.MustGetHostname()
	TestDeviceObtSupportsTestProperties = checkDpsResourceForTestProperties()

	TestDevsimResources = []schema.ResourceLink{
		{
			Href:          platform.ResourceURI,
			ResourceTypes: []string{platform.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          device.ResourceURI,
			ResourceTypes: []string{types.DEVICE_CLOUD, device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          configuration.ResourceURI,
			ResourceTypes: []string{configuration.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          LightResourceInstanceHref("1"),
			ResourceTypes: []string{types.CORE_LIGHT},
			Interfaces:    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},

		{
			Href:          TestResourceSwitchesHref,
			ResourceTypes: []string{collection.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_LL, interfaces.OC_IF_CREATE, interfaces.OC_IF_B, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},
		{
			Href:          ResourcePlgdDpsHref,
			ResourceTypes: []string{ResourcePlgdDpsType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			Policy: &schema.Policy{
				BitMask: 3,
			},
		},
	}
}

func MakeAPIsConfig() service.APIsConfig {
	var cfg service.APIsConfig
	cfg.COAP.Addr = DPSHost
	cfg.COAP.MaxMessageSize = 256 * 1024
	cfg.COAP.MessagePoolSize = 1000
	cfg.COAP.Protocols = []pkgCoapService.Protocol{pkgCoapService.TCP}
	if config.DPS_UDP_ENABLED {
		cfg.COAP.Protocols = append(cfg.COAP.Protocols, pkgCoapService.UDP)
	}
	cfg.COAP.InactivityMonitor = &pkgCoapService.InactivityMonitor{
		Timeout: time.Second * 20,
	}
	cfg.COAP.BlockwiseTransfer.Enabled = config.DPS_UDP_ENABLED
	cfg.COAP.BlockwiseTransfer.SZX = "1024"
	cfg.HTTP = MakeHTTPConfig()
	tlsServerCfg := config.MakeTLSServerConfig()
	cfg.COAP.TLS.Embedded.CertFile = tlsServerCfg.CertFile
	cfg.COAP.TLS.Embedded.KeyFile = tlsServerCfg.KeyFile
	return cfg
}

func MakeConfig(t require.TestingT) service.Config {
	var cfg service.Config
	cfg.Log.DumpBody = true
	cfg.Log = log.MakeDefaultConfig()
	cfg.APIs = MakeAPIsConfig()
	cfg.Clients.Storage = MakeStorageConfig()
	cfg.Clients.OpenTelemetryCollector = pkgHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}
	cfg.EnrollmentGroups = append(cfg.EnrollmentGroups, MakeEnrollmentGroup())
	err := cfg.Validate()
	require.NoError(t, err)

	return cfg
}

func MakeEnrollmentGroup() service.EnrollmentGroupConfig {
	var cfg service.EnrollmentGroupConfig
	cfg.ID = DPSEnrollmentGroupID
	cfg.Owner = DPSOwner
	cfg.AttestationMechanism.X509.CertificateChain = urischeme.URIScheme(os.Getenv("TEST_DPS_INTERMEDIATE_CA_CERT"))
	cfg.Hubs = []service.HubConfig{MakeHubConfig(config.HubID(), config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST)}
	return cfg
}

func MakeAuthorizationConfig() service.AuthorizationConfig {
	var cfg service.AuthorizationConfig
	authCfg := config.MakeAuthorizationConfig()

	cfg.OwnerClaim = "sub"
	cfg.Provider.Name = config.DEVICE_PROVIDER
	cfg.Provider.Config.Authority = authCfg.Endpoints[0].Authority
	cfg.Provider.Config.Audience = authCfg.Audience
	cfg.Provider.Config.HTTP = authCfg.Endpoints[0].HTTP
	cfg.Provider.Config.ClientID = config.OAUTH_MANAGER_CLIENT_ID
	cfg.Provider.Config.ClientSecretFile = config.CA_POOL
	return cfg
}

func MakeHubConfig(hubID string, gateways ...string) service.HubConfig {
	var cfg service.HubConfig
	cfg.Authorization = MakeAuthorizationConfig()
	cfg.CertificateAuthority.Connection = config.MakeGrpcClientConfig(config.CERTIFICATE_AUTHORITY_HOST)
	cfg.Gateways = gateways
	cfg.HubID = hubID
	return cfg
}

func SetUp(t *testing.T) (tearDown func()) {
	return New(t, MakeConfig(t))
}

func New(t *testing.T, cfg service.Config, opts ...service.Option) func() {
	return NewWithContext(context.Background(), t, cfg, opts...)
}

// New creates test dps-gateway.
func NewWithContext(ctx context.Context, t *testing.T, cfg service.Config, opts ...service.Option) func() {
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := service.New(ctx, cfg, fileWatcher, logger, opts...)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	return func() {
		_ = s.Close()
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}
}

func MakeStorageConfig() service.StorageConfig {
	return service.StorageConfig{
		CacheExpiration: time.Second,
		MongoDB: storeMongo.Config{
			Mongo: mongodb.Config{
				MaxPoolSize:     16,
				MaxConnIdleTime: time.Minute * 4,
				URI:             config.MONGODB_URI,
				Database:        "deviceProvisioning",
				TLS:             config.MakeTLSClientConfig(),
			},
			BulkWrite: storeMongo.BulkWriteConfig{
				Timeout:       time.Minute,
				ThrottleTime:  time.Millisecond * 500,
				DocumentLimit: 1000,
			},
		},
	}
}

func MakeHTTPConfig() service.HTTPConfig {
	return service.HTTPConfig{
		Enabled: true,
		Config: http.Config{
			Connection: config.MakeListenerConfig(DPSHTTPHost),
			Authorization: http.AuthorizationConfig{
				OwnerClaim: "sub",
				Config:     config.MakeValidatorConfig(),
			},
			Server: config.MakeHttpServerConfig(),
		},
	}
}

func NewMongoStore(t require.TestingT) (*storeMongo.Store, func()) {
	cfg := MakeConfig(t)
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	certManager, err := cmClient.New(cfg.Clients.Storage.MongoDB.Mongo.TLS, fileWatcher, logger)
	require.NoError(t, err)

	ctx := context.Background()
	store, err := storeMongo.NewStore(ctx, cfg.Clients.Storage.MongoDB, certManager.GetTLSConfig(), logger, noop.NewTracerProvider())
	require.NoError(t, err)

	cleanUp := func() {
		err := store.Clear(ctx)
		require.NoError(t, err)
		_ = store.Close(ctx)
		certManager.Close()

		err = fileWatcher.Close()
		require.NoError(t, err)
	}

	return store, cleanUp
}

func NewHTTPService(ctx context.Context, t *testing.T, store *storeMongo.Store) (*http.Service, func()) {
	cfg := MakeConfig(t)
	cfg.APIs.HTTP.Connection.TLS.ClientCertificateRequired = false
	logger := log.NewLogger(cfg.Log)

	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	s, err := http.New(ctx, "dps-http", cfg.APIs.HTTP.Config, fileWatcher, logger, noop.NewTracerProvider(), store)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()

	cleanUp := func() {
		err = s.Close()
		require.NoError(t, err)
		wg.Wait()
		err = fileWatcher.Close()
		require.NoError(t, err)
	}

	return s, cleanUp
}

func ClearDB(t require.TestingT) {
	_, shutdown := NewMongoStore(t)
	defer shutdown()
}

func NewHub(id, owner string) *pb.Hub {
	return &store.Hub{
		Owner:    owner,
		Id:       id,
		HubId:    id,
		Gateways: []string{"coaps+tcp://1234"},
		Name:     "name",
		CertificateAuthority: &pb.GrpcClientConfig{
			Grpc: &pb.GrpcConnectionConfig{
				Address: "1234",
				Tls: &pb.TlsConfig{
					UseSystemCaPool: true,
				},
			},
		},
		Authorization: &pb.AuthorizationConfig{
			OwnerClaim: "sub",
			Provider: &pb.AuthorizationProviderConfig{
				Name:         "plgd",
				Authority:    "authority",
				ClientId:     "clientId",
				ClientSecret: os.Getenv("TEST_ROOT_CA_CERT"),
				Http: &pb.HttpConfig{
					Tls: &pb.TlsConfig{
						UseSystemCaPool: true,
					},
				},
			},
		},
	}
}

func NewEnrollmentGroup(t *testing.T, id, owner string) *pb.EnrollmentGroup {
	certs, err := security.LoadX509(os.Getenv("TEST_DPS_INTERMEDIATE_CA_CERT"))
	require.NoError(t, err)
	require.NotEmpty(t, certs)

	return &pb.EnrollmentGroup{
		Id:    id,
		Owner: owner,
		AttestationMechanism: &pb.AttestationMechanism{
			X509: &pb.X509Configuration{
				CertificateChain:    os.Getenv("TEST_DPS_INTERMEDIATE_CA_CERT"),
				LeadCertificateName: certs[0].Subject.CommonName,
			},
		},
		HubIds:       []string{id},
		PreSharedKey: "data:,1234567890123456",
		Name:         "name",
	}
}
