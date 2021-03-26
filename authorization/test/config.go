package test

import (
	"testing"

	"github.com/plgd-dev/cloud/authorization/service"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/stretchr/testify/require"
)

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config

	authCfg.Service.GRPC = config.MakeGrpcServerConfig(config.AUTH_HOST)
	authCfg.Service.HTTP = config.MakeListenerConfig(config.AUTH_HTTP_HOST)
	authCfg.Service.HTTP.TLS.ClientCertificateRequired = false

	authCfg.Clients.OAuthClients.Device.Provider = "plgd"
	authCfg.Clients.OAuthClients.Device.OwnerClaim = config.OWNER_CLAIM
	authCfg.Clients.OAuthClients.Device.ClientID = oauthService.ClientTest
	authCfg.Clients.OAuthClients.Device.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.Clients.OAuthClients.Device.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.OAuthClients.Device.HTTP = config.MakeHttpClientConfig()

	authCfg.Clients.OAuthClients.SDK.ClientID = oauthService.ClientTest
	authCfg.Clients.OAuthClients.SDK.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.OAuthClients.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	authCfg.Clients.OAuthClients.SDK.HTTP = config.MakeHttpClientConfig()

	authCfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	authCfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	authCfg.Clients.Storage.MongoDB.Database = "ownersDevices"

	err := authCfg.Validate()
	require.NoError(t, err)
	return authCfg
}
