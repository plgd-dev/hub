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

func SetUpCloudWithConnector(t *testing.T) (TearDown func()) {
	authCfg := authService.MakeConfig(t)
	authCfg.Addr = AUTH_HOST
	authCfg.HTTPAddr = AUTH_HTTP_HOST
	authCfg.MongoDB.Database = cloudConnectorDB
	authShutdown := authService.New(t, authCfg)

	raCfg := raService.MakeConfig(t)
	raCfg.MongoDB.DatabaseName = cloudConnectorDB
	raCfg.Service.Addr = RESOURCE_AGGREGATE_HOST
	raCfg.Service.AuthServerAddr = AUTH_HOST
	raCfg.Nats.URL = cloudConnectorNatsURL
	raShutdown := raService.New(t, raCfg)

	rdCfg := rdService.MakeConfig(t)
	rdCfg.Addr = RESOURCE_DIRECTORY_HOST
	rdCfg.JwksURL = JWKS_URL
	rdCfg.Mongo.DatabaseName = cloudConnectorDB
	rdCfg.Nats.URL = cloudConnectorNatsURL
	rdCfg.Service.AuthServerAddr = AUTH_HOST
	rdCfg.Service.OAuth.Endpoint.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	rdCfg.Service.ResourceAggregateAddr = RESOURCE_AGGREGATE_HOST
	rdCfg.Listen.DisableVerifyClientCertificate = true
	rdShutdown := rdService.New(t, rdCfg)

	c2cConnectorCfg := MakeConfig(t)
	c2cConnectorCfg.StoreMongoDB.DatabaseName = cloudConnectorDB
	c2cConnectorCfg.Service.Addr = C2C_CONNECTOR_HOST
	c2cConnectorCfg.Service.AuthServerAddr = AUTH_HOST
	c2cConnectorCfg.Service.OAuth.Endpoint.TokenURL = OAUTH_MANAGER_ENDPOINT_TOKENURL
	c2cConnectorCfg.Service.OAuthCallback = C2C_CONNECTOR_OAUTH_CALLBACK
	c2cConnectorCfg.Service.EventsURL = C2C_CONNECTOR_EVENTS_URL
	c2cConnectorCfg.Service.ResourceAggregateAddr = RESOURCE_AGGREGATE_HOST
	c2cConnectorCfg.Service.ResourceDirectoryAddr = RESOURCE_DIRECTORY_HOST
	c2cConnectorCfg.Service.JwksURL = JWKS_URL
	c2cConnectorShutdown := New(t, c2cConnectorCfg)

	return func() {
		c2cConnectorShutdown()
		rdShutdown()
		raShutdown()
		authShutdown()
	}
}
