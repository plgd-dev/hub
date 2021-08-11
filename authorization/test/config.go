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
	var cfg service.Config

	cfg.APIs.GRPC = config.MakeGrpcServerConfig(config.AUTH_HOST)
	cfg.APIs.HTTP = config.MakeListenerConfig(config.AUTH_HTTP_HOST)
	cfg.APIs.HTTP.TLS.ClientCertificateRequired = false

	cfg.OAuthClients.Device.Provider = "plgd"
	cfg.OAuthClients.Device.ClientID = oauthService.ClientTest
	cfg.OAuthClients.Device.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	cfg.OAuthClients.Device.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	cfg.OAuthClients.Device.Audience = config.OAUTH_MANAGER_AUDIENCE
	cfg.OAuthClients.Device.HTTP = config.MakeHttpClientConfig()

	cfg.OAuthClients.SDK.ClientID = oauthService.ClientTest
	cfg.OAuthClients.SDK.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	cfg.OAuthClients.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	cfg.OAuthClients.SDK.HTTP = config.MakeHttpClientConfig()

	cfg.Clients.Storage.OwnerClaim = config.OWNER_CLAIM
	cfg.Clients.Storage.MongoDB.URI = config.MONGODB_URI
	cfg.Clients.Storage.MongoDB.TLS = config.MakeTLSClientConfig()
	cfg.Clients.Storage.MongoDB.Database = "ownersDevices"

	cfg.Clients.Eventbus.NATS = config.MakePublisherConfig()

	err := cfg.Validate()
	require.NoError(t, err)
	return cfg
}
