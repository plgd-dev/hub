package service_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	grpcServer "github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
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
		cfg     service.StorageConfig
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     test.MakeStorageConfig(),
			wantErr: false,
		},
		{
			name: "valid - no cron",
			cfg: func() service.StorageConfig {
				cfg := test.MakeStorageConfig()
				cfg.CleanUpRecords = ""
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "invalid - no storage",
			cfg: func() service.StorageConfig {
				cfg := test.MakeStorageConfig()
				cfg.Embedded = storeConfig.Config{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - bad cron expression",
			cfg: func() service.StorageConfig {
				cfg := test.MakeStorageConfig()
				cfg.CleanUpRecords = "bad"
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

func TestResourceAggregateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     service.ResourceAggregateConfig
		wantErr bool
	}{
		{
			name: "valid",
			cfg: service.ResourceAggregateConfig{
				Connection:                   config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST),
				PendingCommandsCheckInterval: time.Second * 10,
			},
		},
		{
			name: "invalid - no connection",
			cfg: func() service.ResourceAggregateConfig {
				cfg := service.ResourceAggregateConfig{
					PendingCommandsCheckInterval: time.Second * 10,
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid - pending commands check interval",
			cfg: func() service.ResourceAggregateConfig {
				cfg := service.ResourceAggregateConfig{
					Connection:                   config.MakeGrpcClientConfig(config.RESOURCE_AGGREGATE_HOST),
					PendingCommandsCheckInterval: -1,
				}
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
				cfg.Storage = service.StorageConfig{}
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
			name: "invalid ResourceAggregate",
			cfg: func() service.ClientsConfig {
				cfg := test.MakeClientsConfig()
				cfg.ResourceAggregate = service.ResourceAggregateConfig{}
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
