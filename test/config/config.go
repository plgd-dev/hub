package config

import (
	"os"
	"time"

	c2curi "github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	grpcClient "github.com/plgd-dev/cloud/pkg/net/grpc/client"
	grpcServer "github.com/plgd-dev/cloud/pkg/net/grpc/server"
	httpClient "github.com/plgd-dev/cloud/pkg/net/http/client"
	"github.com/plgd-dev/cloud/pkg/net/listener"
	"github.com/plgd-dev/cloud/pkg/security/certManager/client"
	"github.com/plgd-dev/cloud/pkg/security/certManager/server"
	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/pkg/security/oauth/manager"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/subscriber"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
)

const AUTH_HOST = "localhost:20000"
const AUTH_HTTP_HOST = "localhost:20001"
const GW_HOST = "localhost:20002"
const RESOURCE_AGGREGATE_HOST = "localhost:20003"
const RESOURCE_DIRECTORY_HOST = "localhost:20004"
const CERTIFICATE_AUTHORITY_HOST = "localhost:20011"
const GRPC_HOST = "localhost:20005"
const C2C_CONNECTOR_HOST = "localhost:20006"
const C2C_GW_HOST = "localhost:20007"
const OAUTH_SERVER_HOST = "localhost:20009"
const TEST_TIMEOUT = time.Second * 20
const OAUTH_MANAGER_CLIENT_ID = service.ClientTest
const OAUTH_MANAGER_AUDIENCE = "localhost"
const HTTP_GW_HOST = "localhost:20010"

var CA_POOL = os.Getenv("LISTEN_FILE_CA_POOL")
var KEY_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_KEY_NAME")
var CERT_FILE = os.Getenv("LISTEN_FILE_CERT_DIR_PATH") + "/" + os.Getenv("LISTEN_FILE_CERT_NAME")
var MONGODB_URI = "mongodb://localhost:27017"
var NATS_URL = "nats://localhost:4222"
var OWNER_CLAIM = "sub"

var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + OAUTH_SERVER_HOST + uri.Authorize
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + OAUTH_SERVER_HOST + uri.Token
var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.Events
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + "/oauthCbk"
var JWKS_URL = "https://" + OAUTH_SERVER_HOST + uri.JWKs

func MakeTLSClientConfig() client.Config {
	return client.Config{
		CAPool:   CA_POOL,
		KeyFile:  KEY_FILE,
		CertFile: CERT_FILE,
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
		Addr:          address,
		TLS:           MakeTLSServerConfig(),
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

func MakePublisherConfig() publisher.Config {
	return publisher.Config{
		URL: NATS_URL,
		TLS: MakeTLSClientConfig(),
	}
}

func MakeSubscriberConfig() subscriber.Config {
	return subscriber.Config{
		URL: NATS_URL,
		TLS: MakeTLSClientConfig(),
	}
}

func MakeEventsStoreMongoDBConfig() mongodb.Config {
	return mongodb.Config{
		URI:             MONGODB_URI,
		BatchSize:       16,
		MaxPoolSize:     16,
		MaxConnIdleTime: 4 * time.Minute,
		Database:        "eventStore",
		TLS:             MakeTLSClientConfig(),
	}
}

func MakeAuthorizationConfig() validator.Config {
	return validator.Config{
		Authority: "https://" + OAUTH_SERVER_HOST,
		Audience:  "https://localhost/",
		HTTP:      MakeHttpClientConfig(),
	}
}

func MakeOAuthConfig() manager.ConfigV2 {
	return manager.ConfigV2{
		ClientID:                    OAUTH_MANAGER_CLIENT_ID,
		ClientSecret:                "secret",
		Audience:                    OAUTH_MANAGER_AUDIENCE,
		TokenURL:                    OAUTH_MANAGER_ENDPOINT_TOKENURL,
		HTTP:                        MakeHttpClientConfig(),
		VerifyServiceTokenFrequency: time.Second * 10,
	}
}
