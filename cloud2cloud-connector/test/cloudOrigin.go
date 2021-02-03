package test

import (
	"testing"

	c2curi "github.com/plgd-dev/cloud/cloud2cloud-connector/uri"
	raService "github.com/plgd-dev/cloud/resource-aggregate/test"
	rdService "github.com/plgd-dev/cloud/resource-directory/test"

	authService "github.com/plgd-dev/cloud/authorization/test"
	"github.com/plgd-dev/cloud/authorization/uri"
)

const AUTH_HOST = "localhost:30000"
const AUTH_HTTP_HOST = "localhost:30001"
const RESOURCE_AGGREGATE_HOST = "localhost:30003"
const RESOURCE_DIRECTORY_HOST = "localhost:30004"
const C2C_CONNECTOR_HOST = "localhost:30006"
const OAUTH_MANAGER_CLIENT_ID = "service"

var C2C_CONNECTOR_EVENTS_URL = "https://" + C2C_CONNECTOR_HOST + c2curi.Events
var C2C_CONNECTOR_OAUTH_CALLBACK = "https://" + C2C_CONNECTOR_HOST + "/oauthCbk"
var OAUTH_MANAGER_ENDPOINT_TOKENURL = "https://" + AUTH_HTTP_HOST + uri.AccessToken
var OAUTH_MANAGER_ENDPOINT_AUTHURL = "https://" + AUTH_HTTP_HOST + uri.AuthorizationCode
var JWKS_URL = "https://" + AUTH_HTTP_HOST + uri.JWKs

const cloudConnectorDB = "cloudConnectorDB"
const cloudConnectorNatsURL = "nats://localhost:34222"
const cloudConnectormongodbURL = "nats://localhost:34223"

func SetUpCloudWithConnector(t *testing.T) (TearDown func()) {
	authCfg := authService.MakeConfig(t)
	authCfg.Service.GrpcServer.GrpcAddr = AUTH_HOST
	authCfg.Service.HttpServer.HttpAddr = AUTH_HTTP_HOST
	authCfg.Database.MongoDB.Database = cloudConnectorDB
	authShutdown := authService.New(t, authCfg)

	raCfg := raService.MakeConfig(t)
	//raCfg.mongodb.URL = cloudConnectormongodbURL
	raCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	raCfg.Service.RA.GrpcAddr = RESOURCE_AGGREGATE_HOST
	raCfg.Clients.AuthServer.AuthServerAddr = AUTH_HOST
	raCfg.Clients.Nats.URL = cloudConnectorNatsURL
	raShutdown := raService.New(t, raCfg)

	rdCfg := rdService.MakeConfig(t)
	rdCfg.Service.RD.GrpcAddr = RESOURCE_DIRECTORY_HOST
	rdCfg.Clients.OAuthProvider.JwksURL = JWKS_URL
	rdCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	//rdCfg.mongodb.URL = cloudConnectormongodbURL
	rdCfg.Clients.Nats.URL = cloudConnectorNatsURL
	rdCfg.Clients.Authorization.Addr = AUTH_HOST
	rdCfg.Clients.OAuthProvider.OAuthConfig.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Clients.ResourceAggregate.Addr = RESOURCE_AGGREGATE_HOST
	rdCfg.Service.RD.GrpcTLSConfig.ClientCertificateRequired = false
	rdShutdown := rdService.New(t, rdCfg)

	c2cConnectorCfg := MakeConfig(t)
	c2cConnectorCfg.Database.MongoDB.DatabaseName = cloudConnectorDB
	c2cConnectorCfg.Service.Http.Addr = C2C_CONNECTOR_HOST
	c2cConnectorCfg.Clients.Authorization.AuthServerAddr = AUTH_HOST
	c2cConnectorCfg.Clients.OAuthProvider.OAuthConfig.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	c2cConnectorCfg.Service.Http.OAuthCallback = C2C_CONNECTOR_OAUTH_CALLBACK
	c2cConnectorCfg.Service.Http.EventsURL = C2C_CONNECTOR_EVENTS_URL
	c2cConnectorCfg.Clients.ResourceAggregate.ResourceAggregateAddr = RESOURCE_AGGREGATE_HOST
	c2cConnectorCfg.Clients.ResourceDirectory.ResourceDirectoryAddr = RESOURCE_DIRECTORY_HOST
	c2cConnectorCfg.Clients.OAuthProvider.JwksURL = JWKS_URL
	c2cConnectorShutdown := New(t, c2cConnectorCfg)

	return func() {
		c2cConnectorShutdown()
		rdShutdown()
		raShutdown()
		authShutdown()
	}
}
