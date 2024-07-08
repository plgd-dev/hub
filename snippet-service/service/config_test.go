package service_test

import (
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/snippet-service/updater"
	"github.com/stretchr/testify/require"
)

func TestAPIsConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     service.APIsConfig
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     test.MakeAPIsConfig(),
			wantErr: false,
		},
		{
			name: "invalid - bad http",
			cfg: func() service.APIsConfig {
				cfg := test.MakeAPIsConfig()
				cfg.HTTP = service.HTTPConfig{
					Addr: "bad",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - no grpc",
			cfg: func() service.APIsConfig {
				cfg := test.MakeAPIsConfig()
				cfg.GRPC = grpcServer.Config{}
				return cfg
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHTTPConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     service.HTTPConfig
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     test.MakeHTTPConfig(),
			wantErr: false,
		},
		{
			name: "invalid - bad address",
			cfg: func() service.HTTPConfig {
				cfg := test.MakeHTTPConfig()
				cfg.Addr = "bad"
				return cfg
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestStorageConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     storeConfig.Config
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     test.MakeStoreConfig(),
			wantErr: false,
		},
		{
			name:    "invalid",
			cfg:     storeConfig.Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClientsConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     service.ClientsConfig
		wantErr bool
	}{
		{
			name: "valid",
			cfg:  test.MakeClientsConfig(),
		},
		{
			name: "invalid - no storage",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.Storage = storeConfig.Config{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid open telemetry",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.OpenTelemetryCollector = otelClient.Config{
					GRPC: otelClient.GRPCConfig{
						Enabled: true,
					},
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid NATS",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.EventBus.NATS = natsClient.Config{
					URL: "bad",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid SubscriptionID",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.EventBus.SubscriptionID = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid ResourceUpdater",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.ResourceUpdater = updater.ResourceUpdaterConfig{}
				return cfg
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     service.Config
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     test.MakeConfig(t),
			wantErr: false,
		},
		{
			name: "invalid - bad log",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Log = log.Config{
					Level: 42,
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - no apis",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs = service.APIsConfig{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - no clients",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients = service.ClientsConfig{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - bad hubID",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.HubID = "bad"
				return cfg
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
