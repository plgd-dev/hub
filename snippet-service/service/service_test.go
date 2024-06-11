// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	httpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	storeCqlDB "github.com/plgd-dev/hub/v2/snippet-service/store/cqldb"
	storeMongo "github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	"github.com/plgd-dev/hub/v2/test/config"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
)

func TestServiceNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	logger := log.NewLogger(log.MakeDefaultConfig())
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)

	const services = hubTestService.SetUpServicesOAuth
	tearDown := hubTestService.SetUpServices(ctx, t, services)
	defer tearDown()

	tests := []struct {
		name    string
		cfg     service.Config
		wantErr bool
	}{
		{
			name: "invalid open telemetry config",
			cfg: service.Config{
				Clients: service.ClientsConfig{
					OpenTelemetryCollector: otelClient.Config{
						GRPC: otelClient.GRPCConfig{
							Enabled: true,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid DB",
			cfg: service.Config{
				Clients: service.ClientsConfig{
					Storage: service.StorageConfig{
						Embedded: storeConfig.Config{
							Use: "invalid",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid mongoDB config",
			cfg: service.Config{
				Clients: service.ClientsConfig{
					Storage: service.StorageConfig{
						Embedded: storeConfig.Config{
							Use: database.MongoDB,
							MongoDB: &storeMongo.Config{
								Mongo: mongodb.Config{
									URI: "invalid",
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid cqlDB config",
			cfg: service.Config{
				Clients: service.ClientsConfig{
					Storage: service.StorageConfig{
						Embedded: storeConfig.Config{
							Use:   database.CqlDB,
							CqlDB: &storeCqlDB.Config{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid CronJob config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients.Storage.CleanUpRecords = "invalid"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid HTTP validator config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs.HTTP.Authorization.Config.HTTP = httpClient.Config{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid HTTP config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs.HTTP.Addr = "invalid"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid GRPC validator config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs.GRPC.Authorization.Config.HTTP = httpClient.Config{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid GRPC config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs.GRPC.Addr = "invalid"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "valid",
			cfg:  test.MakeConfig(t),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := service.New(ctx, tt.cfg, fileWatcher, logger)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			fmt.Printf("cfg: %v\n", tt.cfg)
			require.NoError(t, err)
			_ = s.Close()
		})
	}
}
