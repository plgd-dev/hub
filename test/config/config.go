package config

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	c2curi "github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	pkgMongo "github.com/plgd-dev/hub/v2/pkg/mongodb"
	grpcClient "github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	httpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	"github.com/plgd-dev/hub/v2/pkg/net/listener"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/server"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2"
	"github.com/plgd-dev/hub/v2/pkg/security/oauth2/oauth"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/stretchr/testify/require"
)

const (
	IDENTITY_STORE_HOST          = "localhost:20000"
	IDENTITY_STORE_DB            = "ownersDevices"
	GW_HOST                      = "localhost:20002"
	RESOURCE_AGGREGATE_HOST      = "localhost:20003"
	RESOURCE_DIRECTORY_HOST      = "localhost:20004"
	CERTIFICATE_AUTHORITY_HOST   = "localhost:20011"
	GRPC_HOST                    = "localhost:20005"
	C2C_CONNECTOR_HOST           = "localhost:20006"
	C2C_CONNECTOR_DB             = "cloud2cloudConnector"
	C2C_GW_HOST                  = "localhost:20007"
	C2C_GW_DB                    = "cloud2cloudGateway"
	OAUTH_SERVER_HOST            = "localhost:20009"
	TEST_TIMEOUT                 = time.Second * 30
	OAUTH_MANAGER_CLIENT_ID      = "test"
	OAUTH_MANAGER_AUDIENCE       = "localhost"
	HTTP_GW_HOST                 = "localhost:20010"
	DEVICE_PROVIDER              = "plgd"
	OPENTELEMETRY_COLLECTOR_HOST = "localhost:55690"
)

var CA_POOL = os.Getenv("LISTEN_FILE_CA_POOL")
var KEY_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_KEY_NAME")
var CERT_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_NAME")
var MONGODB_URI = "mongodb://localhost:27017"
var NATS_URL = "nats://localhost:4222"
var OWNER_CLAIM = "sub"

var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + OAUTH_SERVER_HOST + uri.Authorize
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + OAUTH_SERVER_HOST + uri.Token
var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.Events
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + c2curi.OAuthCallback

func MakeTLSClientConfig() client.Config {
	return client.Config{
		CAPool:   CA_POOL,
		KeyFile:  KEY_FILE,
		CertFile: CERT_FILE,
	}
}

func MakeOpenTelemetryCollectorClient() otelClient.Config {
	return otelClient.Config{
		GRPC: otelClient.GRPCConfig{
			Enabled:    true,
			Connection: MakeGrpcClientConfig(OPENTELEMETRY_COLLECTOR_HOST),
		},
	}
}

func MakeGrpcClientConfig(address string) grpcClient.Config {
	return grpcClient.Config{
		Addr: address,
		TLS:  MakeTLSClientConfig(),
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

func MakeGrpcServerConfig(address string) grpcServer.Config {
	return grpcServer.Config{
		Addr: address,
		TLS:  MakeTLSServerConfig(),
		Authorization: grpcServer.AuthorizationConfig{
			OwnerClaim: OWNER_CLAIM,
			Config:     MakeAuthorizationConfig(),
		},
		EnforcementPolicy: grpcServer.EnforcementPolicyConfig{
			MinTime:             time.Second * 5,
			PermitWithoutStream: true,
		},
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

func MakePublisherConfig() natsClient.ConfigPublisher {
	return natsClient.ConfigPublisher{
		Config: natsClient.Config{
			URL:            NATS_URL,
			TLS:            MakeTLSClientConfig(),
			FlusherTimeout: time.Second * 30,
		},
	}
}

func MakeSubscriberConfig() natsClient.Config {
	return natsClient.Config{
		URL: NATS_URL,
		PendingLimits: natsClient.PendingLimitsConfig{
			MsgLimit:   524288,
			BytesLimit: 67108864,
		},
		TLS: MakeTLSClientConfig(),
	}
}

func MakeEventsStoreMongoDBConfig() mongodb.Config {
	return mongodb.Config{
		Embedded: pkgMongo.Config{
			MaxPoolSize:     16,
			MaxConnIdleTime: 4 * time.Minute,
			URI:             MONGODB_URI,
			Database:        "eventStore",
			TLS:             MakeTLSClientConfig(),
		},
	}
}

func MakeAuthorizationConfig() validator.Config {
	return validator.Config{
		Authority: "https://" + OAUTH_SERVER_HOST,
		Audience:  "https://" + OAUTH_MANAGER_AUDIENCE,
		HTTP:      MakeHttpClientConfig(),
	}
}

func MakeDeviceAuthorization() oauth2.Config {
	return oauth2.Config{
		Authority: "https://" + OAUTH_SERVER_HOST,
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
	return "https://" + OAUTH_SERVER_HOST + uri.Authorize
}

const JWTSecret = "secret"

func CreateJwtToken(t *testing.T, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(JWTSecret))
	require.NoError(t, err)
	return tokenString
}
