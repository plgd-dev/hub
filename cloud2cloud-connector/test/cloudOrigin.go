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
	authCfg.Service.GRPC.Addr = AUTH_HOST
	authCfg.Service.HTTP.Addr = AUTH_HTTP_HOST
	authCfg.Databases.MongoDB.Database = cloudConnectorDB
	authShutdown := authService.New(t, authCfg)

	raCfg := raService.MakeConfig(t)
	//raCfg.mongodb.URL = cloudConnectormongodbURL
	raCfg.MongoDB.DatabaseName = cloudConnectorDB
	raCfg.Service.Addr = RESOURCE_AGGREGATE_HOST
	raCfg.Service.AuthServerAddr = AUTH_HOST
	raCfg.Nats.URL = cloudConnectorNatsURL
	raShutdown := raService.New(t, raCfg)

	rdCfg := rdService.MakeConfig(t)
	rdCfg.Addr = RESOURCE_DIRECTORY_HOST
	rdCfg.JwksURL = JWKS_URL
	rdCfg.MongoDB.DatabaseName = cloudConnectorDB
	//rdCfg.mongodb.URL = cloudConnectormongodbURL
	rdCfg.Nats.URL = cloudConnectorNatsURL
	rdCfg.Service.AuthServerAddr = AUTH_HOST
	rdCfg.Service.OAuth.Endpoint.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Service.OAuth.ClientID = OAUTH_MANAGER_CLIENT_ID
	rdCfg.Service.OAuth.Audience = OAUTH_MANAGER_AUDIENCE
	rdShutdown := rdService.New(t, rdCfg)

	grpcCfg := grpcService.MakeConfig(t)
	grpcCfg.Addr = GRPC_GATEWAY_HOST
	grpcCfg.JwksURL = JWKS_URL
	grpcCfg.Nats.URL = cloudConnectorNatsURL
	grpcCfg.Listen.File.DisableVerifyClientCertificate = true
	grpcCfg.ResourceAggregateAddr = RESOURCE_AGGREGATE_HOST
	grpcCfg.ResourceDirectoryAddr = RESOURCE_DIRECTORY_HOST
	grpcShutdown := grpcService.New(t, grpcCfg)

	c2cConnectorCfg := MakeConfig(t)
	c2cConnectorCfg.StoreMongoDB.DatabaseName = cloudConnectorDB
	c2cConnectorCfg.Service.Addr = C2C_CONNECTOR_HOST
	c2cConnectorCfg.Service.AuthServerAddr = AUTH_HOST
	c2cConnectorCfg.Service.OAuth.Endpoint.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	c2cConnectorCfg.Service.OAuth.ClientID = OAUTH_MANAGER_CLIENT_ID
	c2cConnectorCfg.Service.OAuth.Audience = OAUTH_MANAGER_AUDIENCE
	c2cConnectorCfg.Service.OAuthCallback = C2C_CONNECTOR_OAUTH_CALLBACK
	c2cConnectorCfg.Service.EventsURL = C2C_CONNECTOR_EVENTS_URL
	c2cConnectorCfg.Service.ResourceAggregateAddr = RESOURCE_AGGREGATE_HOST
	c2cConnectorCfg.Service.ResourceDirectoryAddr = RESOURCE_DIRECTORY_HOST
	c2cConnectorCfg.Service.JwksURL = JWKS_URL
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
