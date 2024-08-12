package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/device/v2/schema"
	c2curi "github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	m2mOauthUri "github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	pkgCqldb "github.com/plgd-dev/hub/v2/pkg/cqldb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	httpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	httpServer "github.com/plgd-dev/hub/v2/pkg/net/http/server"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/cqldb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/test/http"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	IDENTITY_STORE_HOST             = "localhost:20000"
	IDENTITY_STORE_DB               = "ownersDevices"
	COAP_GW_HOST                    = "localhost:20002"
	RESOURCE_AGGREGATE_HOST         = "localhost:20003"
	RESOURCE_DIRECTORY_HOST         = "localhost:20004"
	CERTIFICATE_AUTHORITY_HOST      = "localhost:20011"
	CERTIFICATE_AUTHORITY_HTTP_HOST = "localhost:20012"
	M2M_OAUTH_SERVER_HTTP_HOST      = "localhost:20013"
	M2M_OAUTH_SERVER_HOST           = "localhost:20016"
	SNIPPET_SERVICE_HOST            = "localhost:20014"
	SNIPPET_SERVICE_HTTP_HOST       = "localhost:20015"
	GRPC_GW_HOST                    = "localhost:20005"
	C2C_CONNECTOR_HOST              = "localhost:20006"
	C2C_CONNECTOR_DB                = "cloud2cloudConnector"
	C2C_GW_HOST                     = "localhost:20007"
	C2C_GW_DB                       = "cloud2cloudGateway"
	OAUTH_SERVER_HOST               = "localhost:20009"
	TEST_TIMEOUT                    = time.Second * 30
	OAUTH_MANAGER_CLIENT_ID         = "test"
	OAUTH_MANAGER_AUDIENCE          = "localhost"
	HTTP_GW_HOST                    = "localhost:20010"
	DEVICE_PROVIDER                 = "plgd"
	OPENTELEMETRY_COLLECTOR_HOST    = "localhost:55690"
	TRUE_STRING                     = "true"
	M2M_OAUTH_PRIVATE_KEY_CLIENT_ID = "JWTPrivateKeyClient"
	VALIDATOR_CACHE_EXPIRATION      = time.Second * 10
)

var (
	CA_POOL                  = urischeme.URIScheme(os.Getenv("LISTEN_FILE_CA_POOL"))
	KEY_FILE                 = urischeme.URIScheme(os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_KEY_NAME"))
	CERT_FILE                = urischeme.URIScheme(os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_NAME"))
	MONGODB_URI              = "mongodb://localhost:27017"
	NATS_URL                 = "nats://localhost:4222"
	OWNER_CLAIM              = "sub"
	COAP_GATEWAY_UDP_ENABLED = os.Getenv("TEST_COAP_GATEWAY_UDP_ENABLED") == TRUE_STRING
	ACTIVE_COAP_SCHEME       = func() string {
		if os.Getenv("TEST_COAP_GATEWAY_UDP_ENABLED") == TRUE_STRING {
			return string(schema.UDPSecureScheme)
		}
		return string(schema.TCPSecureScheme)
	}()
	SCYLLA_HOSTS    = []string{"127.0.0.1"}
	SCYLLA_PORT     = pkgCqldb.DefaultPort
	ACTIVE_DATABASE = func() database.DBUse {
		if database.DBUse(os.Getenv("TEST_DATABASE")).ToLower() == database.CqlDB.ToLower() {
			return database.CqlDB
		}
		return database.MongoDB
	}
)

var (
	OAUTH_MANAGER_ENDPOINT_AUTHURL  = http.HTTPS_SCHEME + OAUTH_SERVER_HOST + uri.Authorize
	OAUTH_MANAGER_ENDPOINT_TOKENURL = http.HTTPS_SCHEME + OAUTH_SERVER_HOST + uri.Token
	C2C_CONNECTOR_EVENTS_URL        = http.HTTPS_SCHEME + C2C_CONNECTOR_HOST + c2curi.Events
	C2C_CONNECTOR_OAUTH_CALLBACK    = http.HTTPS_SCHEME + C2C_CONNECTOR_HOST + c2curi.OAuthCallback
)

func MakeTLSClientConfig() client.Config {
	cfg := client.Config{
		CAPool:   CA_POOL,
		KeyFile:  KEY_FILE,
		CertFile: CERT_FILE,
	}
	_ = cfg.Validate()
	return cfg
}

func MakeOpenTelemetryCollectorClient() otelClient.Config {
	return otelClient.Config{
		GRPC: otelClient.GRPCConfig{
			Enabled:    false,
			Connection: MakeGrpcClientConfig(OPENTELEMETRY_COLLECTOR_HOST),
		},
	}
}

const DefaultGrpcMaxMsgSize = 1024 * 1024 * 128

func MakeGrpcClientConfig(address string) grpcClient.Config {
	return grpcClient.Config{
		Addr:        address,
		SendMsgSize: DefaultGrpcMaxMsgSize,
		RecvMsgSize: DefaultGrpcMaxMsgSize,
		TLS:         MakeTLSClientConfig(),
		KeepAlive: grpcClient.KeepAliveConfig{
			Time:                time.Second * 10,
			Timeout:             time.Second * 20,
			PermitWithoutStream: true,
		},
	}
}

func MakeTLSServerConfig() server.Config {
	return server.Config{
		CAPool:                    CA_POOL,
		KeyFile:                   KEY_FILE,
		CertFile:                  CERT_FILE,
		ClientCertificateRequired: true,
	}
}

func MakeAuthorizationConfig() grpcServer.AuthorizationConfig {
	return grpcServer.AuthorizationConfig{
		OwnerClaim: OWNER_CLAIM,
		Config:     MakeValidatorConfig(),
	}
}

func MakeGrpcServerBaseConfig(address string) grpcServer.BaseConfig {
	return grpcServer.BaseConfig{
		Addr:        address,
		SendMsgSize: DefaultGrpcMaxMsgSize,
		RecvMsgSize: DefaultGrpcMaxMsgSize,
		TLS:         MakeTLSServerConfig(),
		EnforcementPolicy: grpcServer.EnforcementPolicyConfig{
			MinTime:             time.Second * 5,
			PermitWithoutStream: true,
		},
	}
}

func MakeGrpcServerConfig(address string) grpcServer.Config {
	return grpcServer.Config{
		BaseConfig:    MakeGrpcServerBaseConfig(address),
		Authorization: MakeAuthorizationConfig(),
	}
}

func MakeListenerConfig(address string) listener.Config {
	return listener.Config{
		Addr: address,
		TLS:  MakeTLSServerConfig(),
	}
}

func MakeHttpClientConfig() httpClient.Config {
	return httpClient.Config{
		MaxIdleConns:        16,
		MaxConnsPerHost:     32,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     time.Second * 30,
		Timeout:             time.Second * 10,
		TLS:                 MakeTLSClientConfig(),
	}
}

func MakeHttpServerConfig() httpServer.Config {
	return httpServer.Config{
		ReadTimeout:       time.Second * 8,
		ReadHeaderTimeout: time.Second * 4,
		WriteTimeout:      time.Second * 16,
		IdleTimeout:       time.Second * 30,
	}
}

func LeadResourceIsEnabled() bool {
	filter := os.Getenv("TEST_LEAD_RESOURCE_TYPE_FILTER")
	regexFilter := os.Getenv("TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER")
	return filter != "" || regexFilter != ""
}

func LeadResourceUseUUID() bool {
	return os.Getenv("TEST_LEAD_RESOURCE_TYPE_USE_UUID") == TRUE_STRING
}

func MakePublisherConfig(t require.TestingT) natsClient.ConfigPublisher {
	cp := natsClient.ConfigPublisher{
		Config: natsClient.Config{
			URL:            NATS_URL,
			TLS:            MakeTLSClientConfig(),
			FlusherTimeout: time.Second * 30,
		},
	}
	filterIn := os.Getenv("TEST_LEAD_RESOURCE_TYPE_FILTER")
	regexFilterIn := os.Getenv("TEST_LEAD_RESOURCE_TYPE_REGEX_FILTER")
	if filterIn == "" && regexFilterIn == "" {
		return cp
	}
	lrt := &natsClient.LeadResourceTypePublisherConfig{
		Enabled: true,
		UseUUID: LeadResourceUseUUID(),
	}
	if filterIn != "" {
		err := natsClient.CheckResourceTypeFilterString(filterIn)
		require.NoError(t, err)
		lrt.Filter = natsClient.LeadResourceTypeFilter(filterIn)
	}
	if regexFilterIn != "" {
		rfs := strings.Split(regexFilterIn, ",")
		for _, rf := range rfs {
			_, err := regexp.Compile(rf)
			require.NoError(t, err)
		}
		lrt.RegexFilter = rfs
	}
	cp.LeadResourceType = lrt
	return cp
}

func MakeSubscriberConfig() natsClient.ConfigSubscriber {
	return natsClient.ConfigSubscriber{
		Config: natsClient.Config{
			URL: NATS_URL,
			PendingLimits: natsClient.PendingLimitsConfig{
				MsgLimit:   524288,
				BytesLimit: 67108864,
			},
			TLS: MakeTLSClientConfig(),
		},
		LeadResourceType: &natsClient.LeadResourceTypeSubscriberConfig{
			Enabled: LeadResourceIsEnabled(),
		},
	}
}

func MakeEventsStoreMongoDBConfig() *mongodb.Config {
	return &mongodb.Config{
		Embedded: pkgMongo.Config{
			MaxPoolSize:     16,
			MaxConnIdleTime: 4 * time.Minute,
			URI:             MONGODB_URI,
			Database:        "eventStore",
			TLS:             MakeTLSClientConfig(),
		},
	}
}

func MakeCqlDBConfig() pkgCqldb.Config {
	return pkgCqldb.Config{
		Hosts:                 SCYLLA_HOSTS,
		TLS:                   MakeTLSClientConfig(),
		NumConns:              16,
		Port:                  SCYLLA_PORT,
		ConnectTimeout:        time.Second * 10,
		UseHostnameResolution: true,
		ReconnectionPolicy: pkgCqldb.ReconnectionPolicyConfig{
			Constant: pkgCqldb.ConstantReconnectionPolicyConfig{
				Interval:   time.Second * 3,
				MaxRetries: 3,
			},
		},
		Keyspace: pkgCqldb.KeyspaceConfig{
			Name:   "plgdhub",
			Create: true,
			Replication: map[string]interface{}{
				"class":              "SimpleStrategy",
				"replication_factor": 1,
			},
		},
	}
}

func MakeEventsStoreCqlDBConfig() *cqldb.Config {
	return &cqldb.Config{
		Table:    "events",
		Embedded: MakeCqlDBConfig(),
	}
}

func MakeValidatorConfig() validator.Config {
	return validator.Config{
		Audience: http.HTTPS_SCHEME + OAUTH_MANAGER_AUDIENCE,
		Endpoints: []validator.AuthorityConfig{
			{
				Authority: http.HTTPS_SCHEME + OAUTH_SERVER_HOST,
				HTTP:      MakeHttpClientConfig(),
			},
			{
				Authority: http.HTTPS_SCHEME + M2M_OAUTH_SERVER_HTTP_HOST + m2mOauthUri.Base,
				HTTP:      MakeHttpClientConfig(),
			},
		},
		TokenVerification: validator.TokenTrustVerificationConfig{
			CacheExpiration: VALIDATOR_CACHE_EXPIRATION,
		},
	}
}

func MakeDeviceAuthorization() oauth2.Config {
	return oauth2.Config{
		Authority: http.HTTPS_SCHEME + OAUTH_SERVER_HOST,
		Config: oauth.Config{
			ClientID:         OAUTH_MANAGER_CLIENT_ID,
			Audience:         OAUTH_MANAGER_AUDIENCE,
			RedirectURL:      "cloud.plgd.mobile://login-callback",
			ClientSecretFile: CA_POOL, // any generated file
		},
		HTTP: MakeHttpClientConfig(),
	}
}

func HubID() string {
	return os.Getenv("TEST_CLOUD_SID")
}

func MakeAuthURL() string {
	return http.HTTPS_SCHEME + OAUTH_SERVER_HOST + uri.Authorize
}

const JWTSecret = "secret"

func CreateJwtToken(t *testing.T, claims jwt.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(JWTSecret))
	require.NoError(t, err)
	return tokenString
}

func MakeLogConfig(t require.TestingT, envLogLevel, envLogDumpBody string) log.Config {
	cfg := log.MakeDefaultConfig()
	logLvlString := os.Getenv(envLogLevel)
	logLvl := zap.NewAtomicLevelAt(log.InfoLevel)
	if logLvlString != "" {
		var err error
		logLvl, err = zap.ParseAtomicLevel(logLvlString)
		require.NoError(t, err)
	}
	cfg.Level = logLvl.Level()
	logDumpBodyStr := strings.ToLower(os.Getenv(envLogDumpBody))
	switch logDumpBodyStr {
	case TRUE_STRING, "false":
		cfg.DumpBody = logDumpBodyStr == TRUE_STRING
	case "":
		cfg.DumpBody = false
	default:
		require.NoError(t, fmt.Errorf("invalid value %v for %v", logDumpBodyStr, envLogDumpBody))
	}
	return cfg
}
