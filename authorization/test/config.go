package test

import (
	"testing"

	"github.com/plgd-dev/cloud/authorization/service"
	"github.com/plgd-dev/cloud/test/config"
	oauthService "github.com/plgd-dev/cloud/test/oauth-server/service"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
)

func MakeConfig(t *testing.T) service.Config {
	var authCfg service.Config

	authCfg.Service.GRPC.Addr = config.AUTH_HOST
	authCfg.Service.GRPC.TLS.CAPool = config.CA_POOL
	authCfg.Service.GRPC.TLS.CertFile = config.CERT_FILE
	authCfg.Service.GRPC.TLS.KeyFile = config.KEY_FILE
	authCfg.Service.HTTP.TLS.ClientCertificateRequired = false

	authCfg.Service.HTTP.Addr = config.AUTH_HTTP_HOST
	authCfg.Service.HTTP.TLS.CAPool = config.CA_POOL
	authCfg.Service.HTTP.TLS.CertFile = config.CERT_FILE
	authCfg.Service.HTTP.TLS.KeyFile = config.KEY_FILE
	authCfg.Service.HTTP.TLS.ClientCertificateRequired = false

	authCfg.Clients.Device.Provider = "plgd"
	authCfg.Clients.Device.OwnerClaim = "sub"
	authCfg.Clients.Device.ClientID = oauthService.ClientTest
	authCfg.Clients.Device.AuthURL = "https://" + config.OAUTH_SERVER_HOST + uri.Authorize
	authCfg.Clients.Device.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.Device.HTTP.TLS.CAPool = config.CA_POOL
	authCfg.Clients.Device.HTTP.TLS.CertFile = config.CERT_FILE
	authCfg.Clients.Device.HTTP.TLS.KeyFile = config.KEY_FILE

	authCfg.Clients.SDK.ClientID = oauthService.ClientTest
	authCfg.Clients.SDK.TokenURL = "https://" + config.OAUTH_SERVER_HOST + uri.Token
	authCfg.Clients.SDK.Audience = config.OAUTH_MANAGER_AUDIENCE
	authCfg.Clients.SDK.HTTP.TLS.CAPool = config.CA_POOL
	authCfg.Clients.SDK.HTTP.TLS.CertFile = config.CERT_FILE
	authCfg.Clients.SDK.HTTP.TLS.KeyFile = config.KEY_FILE

	authCfg.Databases.MongoDB.URI = config.MONGODB_URI
	authCfg.Databases.MongoDB.TLS.CAPool = config.CA_POOL
	authCfg.Databases.MongoDB.TLS.CertFile = config.CERT_FILE
	authCfg.Databases.MongoDB.TLS.KeyFile = config.KEY_FILE
	return authCfg
}
