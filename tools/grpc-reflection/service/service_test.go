package service_test

import (
	"testing"

	certAuthorityPb "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcGatewayPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	m2mOAuthServerPb "github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	snippetServicePb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/tools/grpc-reflection/service"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	config := service.Config{
		APIs: service.APIsConfig{
			GRPC: service.GRPCConfig{
				ReflectedServices: []string{
					grpcGatewayPb.GrpcGateway_ServiceDesc.ServiceName,
					certAuthorityPb.CertificateAuthority_ServiceDesc.ServiceName,
					snippetServicePb.SnippetService_ServiceDesc.ServiceName,
					m2mOAuthServerPb.M2MOAuthService_ServiceDesc.ServiceName,
				},
				BaseConfig: config.MakeGrpcServerBaseConfig("0.0.0.0:0"),
			},
		},
	}
	err := config.Validate()
	require.NoError(t, err)

	str := config.String()
	require.NotEmpty(t, str)

	fileWatcher, err := fsnotify.NewWatcher(log.Get())
	require.NoError(t, err)

	svc, err := service.New(config, fileWatcher, log.Get())
	require.NoError(t, err)
	require.NotNil(t, svc)
	err = svc.Close()
	require.NoError(t, err)
}
