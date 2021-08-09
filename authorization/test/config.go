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

	authCfg.APIs.GRPC = config.MakeGrpcServerConfig(config.AUTH_HOST)
	authCfg.APIs.HTTP = config.MakeListenerConfig(config.AUTH_HTTP_HOST)
	authCfg.APIs.HTTP.TLS.ClientCertificateRequired = false

	authCfg.OAuthClients.Device.Provider = "plgd"
	authCfg.OAuthClients.Device.ClientID = oauthService.ClientTest
	authCfg.OAuthClients.Device.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.OAuthClients.Device.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.OAuthClients.Device.Audience = config.OAUTH_MANAGER_AUDIENCE
	authCfg.OAuthClients.Device.HTTP = config.MakeHttpClientConfig()

	authCfg.OAuthClients.SDK.ClientID = oauthService.ClientTest
	authCfg.OAuthClients.SDK.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.OAuthClients.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	authCfg.OAuthClients.SDK.HTTP = config.MakeHttpClientConfig()

	authCfg.Clients.Storage.OwnerClaim = config.OWNER_CLAIM
	authCfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	authCfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	authCfg.Clients.Storage.MongoDB.Database = "ownersDevices"

	err := authCfg.Validate()
	require.NoError(t, err)
	return authCfg
}
