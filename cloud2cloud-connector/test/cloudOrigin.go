package test

import (
	"testing"

	c2curi "github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	grpcService "github.com/plgd-dev/cloud/grpc-gateway/test"
	raService "github.com/plgd-dev/cloud/resource-aggregate/test"
	rdService "github.com/plgd-dev/cloud/resource-directory/test"

	authService "github.com/plgd-dev/cloud/authorization/test"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/cloud/test/oauth-server/test"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
)

const AUTH_HOST = "localhost:30000"
const AUTH_HTTP_HOST = "localhost:30001"
const RESOURCE_AGGREGATE_HOST = "localhost:30003"
const RESOURCE_DIRECTORY_HOST = "localhost:30004"
const C2C_CONNECTOR_HOST = "localhost:30006"
const OAUTH_HOST = "localhost:30007"
const GRPC_GATEWAY_HOST = "localhost:30008"
const OAUTH_MANAGER_CLIENT_ID = oauthService.ClientTest
const OAUTH_MANAGER_AUDIENCE = "localhost"

var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.Events
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + "/oauthCbk"
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + OAUTH_HOST + uri.Token
var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + OAUTH_HOST + uri.Authorize
var JWKS_URL = "https://" + OAUTH_HOST + uri.JWKs

const cloudConnectorDB = "cloudConnectorDB"
const cloudConnectorNatsURL = "nats://localhost:34222"
const cloudConnectormongodbURL = "nats://localhost:34223"

func SetUpCloudWithConnector(t *testing.T) (TearDown func()) {
	oauthCfg := oauthTest.MakeConfig(t)
	oauthCfg.Address = OAUTH_HOST
	oauthShutdown := oauthTest.New(t, oauthCfg)

	authCfg := authService.MakeConfig(t)
	authCfg.Service.Grpc.Addr = AUTH_HOST
	authCfg.Service.Http.Addr = AUTH_HTTP_HOST
	authCfg.Database.MongoDB.Database = cloudConnectorDB
	authShutdown := authService.New(t, authCfg)

	raCfg := raService.MakeConfig(t)
	//raCfg.mongodb.URL = cloudConnectormongodbURL
	raCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	raCfg.Service.Grpc.Addr = RESOURCE_AGGREGATE_HOST
	raCfg.Clients.Authorization.Addr = AUTH_HOST
	raCfg.Clients.Nats.URL = cloudConnectorNatsURL
	raShutdown := raService.New(t, raCfg)

	rdCfg := rdService.MakeConfig(t)
	rdCfg.Service.Grpc.Addr = RESOURCE_DIRECTORY_HOST
	rdCfg.Clients.OAuthProvider.JwksURL = JWKS_URL
	rdCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	//rdCfg.mongodb.URL = cloudConnectormongodbURL

	rdCfg.Clients.Nats.URL = cloudConnectorNatsURL
	rdCfg.Clients.Authorization.Addr = AUTH_HOST
	rdCfg.Clients.OAuthProvider.OAuth.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Clients.OAuthProvider.OAuth.ClientID = OAUTH_MANAGER_CLIENT_ID
	rdCfg.Clients.OAuthProvider.OAuth.Audience = OAUTH_MANAGER_AUDIENCE
	rdShutdown := rdService.New(t, rdCfg)

	grpcCfg := grpcService.MakeConfig(t)
	grpcCfg.Service.GrpcConfig.Addr = GRPC_GATEWAY_HOST
	grpcCfg.Service.GrpcConfig.TLSConfig.ClientCertificateRequired = false
	grpcCfg.Clients.OAuthProvider.JwksURL = JWKS_URL
	grpcCfg.Clients.Nats.URL = cloudConnectorNatsURL
	grpcCfg.Clients.ResourceAggregate.Addr = RESOURCE_AGGREGATE_HOST
	grpcCfg.Clients.ResourceDirectory.Addr = RESOURCE_DIRECTORY_HOST
	grpcShutdown := grpcService.New(t, grpcCfg)

	c2cConnectorCfg := MakeConfig(t)
	c2cConnectorCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	c2cConnectorCfg.Service.Http.Addr = C2C_CONNECTOR_HOST
	c2cConnectorCfg.Clients.Authorization.Addr = AUTH_HOST
	c2cConnectorCfg.Clients.OAuthProvider.OAuth.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	c2cConnectorCfg.Clients.OAuthProvider.OAuth.ClientID = OAUTH_MANAGER_CLIENT_ID
	c2cConnectorCfg.Clients.OAuthProvider.OAuth.Audience = OAUTH_MANAGER_AUDIENCE

	c2cConnectorCfg.Service.Http.OAuthCallback = C2C_CONNECTOR_OAUTH_CALLBACK
	c2cConnectorCfg.Service.Http.EventsURL = C2C_CONNECTOR_EVENTS_URL
	c2cConnectorCfg.Clients.ResourceAggregate.Addr = RESOURCE_AGGREGATE_HOST
	c2cConnectorCfg.Clients.ResourceDirectory.Addr = RESOURCE_DIRECTORY_HOST
	c2cConnectorCfg.Clients.OAuthProvider.JwksURL = JWKS_URL
	c2cConnectorShutdown := New(t, c2cConnectorCfg)

	return func() {
		c2cConnectorShutdown()
		grpcShutdown()
		rdShutdown()
		raShutdown()
		authShutdown()
		oauthShutdown()
	}
}
