package service_test

import (
	"testing"

	certAuthorityPb "github.com/plgd-dev/hub/v2/certificate-authority/pb"
	grpcGatewayPb "github.com/plgd-dev/hub/v2/grpc-gateway/pb"
	"github.com/plgd-dev/hub/v2/pkg/log"
	snippetServicePb "github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/plgd-dev/hub/v2/tools/grpc-reflection/service"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	type fields struct {
		Log  log.Config
		APIs service.APIsConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "valid",
			fields: fields{
				Log: log.Config{
					// Fill in the required fields for a valid log.Config instance
				},
				APIs: service.APIsConfig{
					GRPC: service.GRPCConfig{
						ReflectedServices: []string{grpcGatewayPb.GrpcGateway_ServiceDesc.ServiceName, certAuthorityPb.CertificateAuthority_ServiceDesc.ServiceName, snippetServicePb.SnippetService_ServiceDesc.ServiceName},
						BaseConfig:        config.MakeGrpcServerBaseConfig("0.0.0.0:0"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid log config",
			fields: fields{
				Log: log.Config{
					Encoding: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid APIs config",
			fields: fields{
				APIs: service.APIsConfig{
					GRPC: service.GRPCConfig{
						ReflectedServices: []string{"invalid"},
						BaseConfig:        config.MakeGrpcServerBaseConfig("0.0.0.0:0"),
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &service.Config{
				Log:  tt.fields.Log,
				APIs: tt.fields.APIs,
			}
			err := c.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
