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
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/plgd-dev/go-coap/v3/message"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	httpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	otelClient "github.com/plgd-dev/hub/v2/pkg/opentelemetry/collector/client"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	natsClient "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/client"
	"github.com/plgd-dev/hub/v2/snippet-service/pb"
	"github.com/plgd-dev/hub/v2/snippet-service/service"
	storeConfig "github.com/plgd-dev/hub/v2/snippet-service/store/config"
	storeCqlDB "github.com/plgd-dev/hub/v2/snippet-service/store/cqldb"
	storeMongo "github.com/plgd-dev/hub/v2/snippet-service/store/mongodb"
	"github.com/plgd-dev/hub/v2/snippet-service/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	hubTestService "github.com/plgd-dev/hub/v2/test/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
			name: "invalid resource subscriber config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients.NATS = natsClient.Config{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid resource aggregate client config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients.ResourceAggregate = service.ResourceAggregateConfig{}
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
			name: "invalid HTTP config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.APIs.HTTP.Addr = "invalid"
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

func TestService(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()

	logCfg := log.MakeDefaultConfig()
	logCfg.Level = log.DebugLevel
	log.Setup(logCfg)
	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	snippetCfg := test.MakeConfig(t)
	shutdownSnippetService := test.New(t, snippetCfg)
	defer shutdownSnippetService()

	snippetClientConn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = snippetClientConn.Close()
	}()
	snippetClient := pb.NewSnippetServiceClient(snippetClientConn)

	token := oauthTest.GetDefaultAccessToken(t)
	ctx = pkgGrpc.CtxWithToken(ctx, token)

	// configuration -> /light/1 -> { state: on, power: 42 }
	conf, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update light",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: hubTest.TestResourceLightInstanceHref("1"),
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"state": true,
						"power": 42,
					}),
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf.GetId())

	// condition for /light/1
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "apply update light",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{hubTest.TestResourceLightInstanceHref("1")},
		ApiAccessToken:     token,
	})
	require.NoError(t, err)

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		err = grpcClient.Close()
		require.NoError(t, err)
	}()

	resources := hubTest.GetAllBackendResourceLinks()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctx, t, grpcClient.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, resources)
	defer shutdownDevSim()

	var got map[interface{}]interface{}
	err = grpcClient.GetResource(ctx, deviceID, hubTest.TestResourceLightInstanceHref("1"), &got)
	require.NoError(t, err)

	require.Equal(t, map[interface{}]interface{}{
		"state": true,
		"power": uint64(42),
		"name":  "Light",
	}, got)

	// restore state
	err = grpcClient.UpdateResource(ctx, deviceID, hubTest.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"state": false,
		"power": uint64(0),
	}, nil)
	require.NoError(t, err)
}
