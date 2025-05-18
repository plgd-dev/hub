package test

import (
	"testing"

	c2curi "github.com/plgd-dev/hub/v2/cloud2cloud-connector/uri"
	grpcService "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	idService "github.com/plgd-dev/hub/v2/identity-store/test"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	kitNetHttp "github.com/plgd-dev/hub/v2/pkg/net/http"
	raService "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	rdService "github.com/plgd-dev/hub/v2/resource-directory/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/test/http"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
)

const (
	IDENTITY_STORE_HOST     = "localhost:30000"
	RESOURCE_AGGREGATE_HOST = "localhost:30003"
	RESOURCE_DIRECTORY_HOST = "localhost:30004"
	C2C_CONNECTOR_HOST      = "localhost:30006"
	OAUTH_HOST              = "localhost:30007"
	GRPC_GATEWAY_HOST       = "localhost:30008"
	C2C_CONNECTOR_DB        = "cloudConnectorDB"
	C2C_CONNECTOR_NATS_URL  = "nats://localhost:34222"
)

var (
	C2C_CONNECTOR_EVENTS_URL        = http.HTTPS_SCHEME + C2C_CONNECTOR_HOST + c2curi.Events
	C2C_CONNECTOR_OAUTH_CALLBACK    = http.HTTPS_SCHEME + C2C_CONNECTOR_HOST + c2curi.OAuthCallback
	OAUTH_MANAGER_ENDPOINT_TOKENURL = http.HTTPS_SCHEME + OAUTH_HOST + uri.Token
)

func SetUpCloudWithConnector(t *testing.T) (tearDown func()) {
	var cleanUp fn.FuncList
	deferedCleanUp := true
	defer func() {
		if deferedCleanUp {
			cleanUp.Execute()
		}
	}()

	oauthCfg := oauthTest.MakeConfig(t)
	oauthCfg.APIs.HTTP.Connection.Addr = OAUTH_HOST
	oauthCfg.OAuthSigner.Domain = OAUTH_HOST
	oauthShutdown := oauthTest.New(t, oauthCfg)
	cleanUp.AddFunc(oauthShutdown)

	idCfg := idService.MakeConfig(t)
	idCfg.APIs.GRPC.Addr = IDENTITY_STORE_HOST
	idCfg.APIs.GRPC.Authorization.Config.Endpoints[0].Authority = http.HTTPS_SCHEME + OAUTH_HOST
	idCfg.Clients.Storage.MongoDB.Database = C2C_CONNECTOR_DB
	idCfg.Clients.Eventbus.NATS.URL = C2C_CONNECTOR_NATS_URL
	idShutdown := idService.New(t, idCfg)
	cleanUp.AddFunc(idShutdown)

	raCfg := raService.MakeConfig(t)
	raCfg.APIs.GRPC.Addr = RESOURCE_AGGREGATE_HOST
	raCfg.APIs.GRPC.Authorization.Config.Endpoints[0].Authority = http.HTTPS_SCHEME + OAUTH_HOST
	raCfg.Clients.Eventstore.Connection.Use = database.MongoDB
	raCfg.Clients.Eventstore.Connection.MongoDB.Embedded.Database = C2C_CONNECTOR_DB
	raCfg.Clients.IdentityStore.Connection.Addr = IDENTITY_STORE_HOST
	raCfg.Clients.Eventbus.NATS.URL = C2C_CONNECTOR_NATS_URL
	raShutdown := raService.New(t, raCfg)
	cleanUp.AddFunc(raShutdown)

	rdCfg := rdService.MakeConfig(t)
	rdCfg.APIs.GRPC.Addr = RESOURCE_DIRECTORY_HOST
	rdCfg.APIs.GRPC.Authorization.Config.Endpoints[0].Authority = http.HTTPS_SCHEME + OAUTH_HOST
	rdCfg.Clients.Eventstore.Connection.Use = database.MongoDB
	rdCfg.Clients.Eventstore.Connection.MongoDB.Embedded.Database = C2C_CONNECTOR_DB
	rdCfg.Clients.Eventbus.NATS.URL = C2C_CONNECTOR_NATS_URL
	rdCfg.Clients.IdentityStore.Connection.Addr = IDENTITY_STORE_HOST
	rdShutdown := rdService.New(t, rdCfg)
	cleanUp.AddFunc(rdShutdown)

	grpcCfg := grpcService.MakeConfig(t)
	grpcCfg.APIs.GRPC.Addr = GRPC_GATEWAY_HOST
	grpcCfg.APIs.GRPC.TLS.ClientCertificateRequired = false
	grpcCfg.APIs.GRPC.Authorization.Config.Endpoints[0].Authority = http.HTTPS_SCHEME + OAUTH_HOST
	grpcCfg.Clients.Eventbus.NATS.URL = C2C_CONNECTOR_NATS_URL
	grpcCfg.Clients.ResourceAggregate.Connection.Addr = RESOURCE_AGGREGATE_HOST
	grpcCfg.Clients.ResourceDirectory.Connection.Addr = RESOURCE_DIRECTORY_HOST
	grpcShutdown := grpcService.New(t, grpcCfg)
	cleanUp.AddFunc(grpcShutdown)

	c2cConnectorCfg := MakeConfig(t)
	c2cConnectorCfg.APIs.HTTP.EventsURL = C2C_CONNECTOR_EVENTS_URL
	c2cConnectorCfg.APIs.HTTP.Connection.Addr = C2C_CONNECTOR_HOST
	c2cConnectorCfg.APIs.HTTP.Authorization = MakeAuthorizationConfig()
	c2cConnectorCfg.APIs.HTTP.Authorization.Authority = http.HTTPS_SCHEME + OAUTH_HOST
	c2cConnectorCfg.APIs.HTTP.Authorization.RedirectURL = C2C_CONNECTOR_OAUTH_CALLBACK
	c2cConnectorCfg.APIs.HTTP.Server = config.MakeHttpServerConfig()
	c2cConnectorCfg.Clients.Storage.MongoDB.Database = C2C_CONNECTOR_DB
	c2cConnectorCfg.Clients.IdentityStore.Connection.Addr = IDENTITY_STORE_HOST
	c2cConnectorCfg.Clients.GrpcGateway.Connection.Addr = GRPC_GATEWAY_HOST
	c2cConnectorCfg.Clients.ResourceAggregate.Connection.Addr = RESOURCE_AGGREGATE_HOST
	c2cConnectorCfg.Clients.Eventbus.NATS.URL = C2C_CONNECTOR_NATS_URL
	c2cConnectorCfg.Clients.OpenTelemetryCollector = kitNetHttp.OpenTelemetryCollectorConfig{
		Config: config.MakeOpenTelemetryCollectorClient(),
	}

	c2cConnectorShutdown := New(t, c2cConnectorCfg)
	cleanUp.AddFunc(c2cConnectorShutdown)

	deferedCleanUp = false
	return cleanUp.ToFunction()
}
