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
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema/configuration"
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
	"github.com/plgd-dev/hub/v2/snippet-service/store"
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
				cfg.Clients.EventBus.NATS = natsClient.Config{}
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
				cfg.APIs.GRPC.Authorization.Config.HTTP = &httpClient.Config{}
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

func getAppliedConfigurations(ctx context.Context, t *testing.T, snippetClient pb.SnippetServiceClient, query *pb.GetAppliedConfigurationsRequest) (map[string]*pb.AppliedConfiguration, map[string]*pb.AppliedConfiguration_Resource) {
	getClient, err := snippetClient.GetAppliedConfigurations(ctx, query)
	require.NoError(t, err)
	defer func() {
		_ = getClient.CloseSend()
	}()
	appliedConfs := make(map[string]*pb.AppliedConfiguration)
	for {
		appliedConf, errR := getClient.Recv()
		if errors.Is(errR, io.EOF) {
			break
		}
		require.NoError(t, errR)
		appliedConfs[appliedConf.GetId()] = appliedConf
	}
	appliedConfResources := make(map[string]*pb.AppliedConfiguration_Resource)
	for _, appliedConf := range appliedConfs {
		for _, r := range appliedConf.GetResources() {
			id := appliedConf.GetConfigurationId().GetId() + "." + r.GetHref()
			appliedConfResources[id] = r
		}
	}
	return appliedConfs, appliedConfResources
}

func TestService(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	snippetCfg := test.MakeConfig(t)
	snippetCfg.Clients.ResourceAggregate.PendingCommandsCheckInterval = time.Millisecond * 500
	_, shutdownSnippetService := test.New(t, snippetCfg)
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

	notExistingResourceHref := "/not/existing"
	// configuration1
	// -> /light/1 -> { state: on }
	// -> /not/existing -> { value: 42 }
	conf1, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: hubTest.TestResourceLightInstanceHref("1"),
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"state": true,
					}),
				},
			},
			{
				Href: notExistingResourceHref,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"value": 42,
					}),
				},
				TimeToLive: int64(100 * time.Millisecond),
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf1.GetId())

	// configuration2 -> /light/1 -> { power: 42 }
	conf2, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update light power",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: hubTest.TestResourceLightInstanceHref("1"),
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"power": 42,
					}),
				},
				TimeToLive: int64(500 * time.Millisecond),
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf2.GetId())

	// configuration3 -> /oc/con-> {n: "updated name"}
	conf3, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update oc/con",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: configuration.ResourceURI,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"n": "updated name",
					}),
				},
				TimeToLive: int64(500 * time.Millisecond),
			},
		},
	})
	require.NoError(t, err)

	// skipped condition for conf1 - missing ApiAccessToken -> will be skipped during evaluation
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "skipped update light state",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf1.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{hubTest.TestResourceLightInstanceHref("1")},
	})
	require.NoError(t, err)

	// valid condition for conf1
	cond1, err := snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "update light state",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf1.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{notExistingResourceHref, hubTest.TestResourceLightInstanceHref("1")},
		ApiAccessToken:     token,
	})
	require.NoError(t, err)

	// invalid condition for conf1 - invalid ApiAccessToken
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "fail update light state",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf1.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{notExistingResourceHref, hubTest.TestResourceLightInstanceHref("1")},
		ApiAccessToken:     "an invalid token",
	})
	require.NoError(t, err)

	// condition for conf2
	cond2, err := snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "update light power",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf2.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{hubTest.TestResourceLightInstanceHref("1")},
		ApiAccessToken:     token,
	})
	require.NoError(t, err)

	// disabled condition for conf3
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "disabled update device name",
		Owner:              oauthService.DeviceUserID,
		Enabled:            false,
		ConfigurationId:    conf3.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{configuration.ResourceURI},
		ApiAccessToken:     token,
	})
	require.NoError(t, err)
	// jq evaluated to false -> non matching name
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "jq evaluated to false",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf3.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{configuration.ResourceURI},
		ApiAccessToken:     token,
		JqExpressionFilter: ".n !== \"" + hubTest.TestDeviceName + "\"",
	})
	require.NoError(t, err)
	// invalid condition for conf3 - invalid ApiAccessToken
	// -> this condition will be tried, but will fail, because of the invalid token,
	// but since no other condition is available, the resource update will be set to failed state
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Name:               "fail update device name",
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf3.GetId(),
		DeviceIdFilter:     []string{deviceID},
		ResourceHrefFilter: []string{configuration.ResourceURI},
		ApiAccessToken:     "an invalid token",
		JqExpressionFilter: ".n == \"" + hubTest.TestDeviceName + "\"",
	})
	require.NoError(t, err)

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		err = grpcClient.Close()
		require.NoError(t, err)
	}()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctx, t, grpcClient.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, hubTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	// -> wait for /conf1 to be applied -> for /not/existing resource this should start-up the timeout timer
	notExistingConf1ID := conf1.GetId() + "." + notExistingResourceHref
	logger := log.NewLogger(snippetCfg.Log)
	var appliedConf1Status pb.AppliedConfiguration_Resource_Status
	retryCount := 0
	for retryCount < 10 {
		logger.Debugf("check conf1 (retry: %v)", retryCount)
		appliedConfs, appliedConfResources := getAppliedConfigurations(ctx, t, snippetClient, &pb.GetAppliedConfigurationsRequest{
			DeviceIdFilter: []string{deviceID},
			ConfigurationIdFilter: []*pb.IDFilter{
				{
					Id: conf1.GetId(),
					Version: &pb.IDFilter_All{
						All: true,
					},
				},
			},
		})
		appliedConf1Status = appliedConfResources[notExistingConf1ID].GetStatus()
		if appliedConf1Status != pb.AppliedConfiguration_Resource_QUEUED {
			logger.Infof("conf1 applied to resources: %v", appliedConfs)
			break
		}
		time.Sleep(time.Millisecond * 200)
		retryCount++
	}
	require.NotEqual(t, pb.AppliedConfiguration_Resource_QUEUED, appliedConf1Status)

	if appliedConf1Status != pb.AppliedConfiguration_Resource_TIMEOUT &&
		appliedConf1Status != pb.AppliedConfiguration_Resource_DONE {
		// -> wait enough time to timeout pending commands
		time.Sleep(2 * snippetCfg.Clients.ResourceAggregate.PendingCommandsCheckInterval)
	}

	var got map[interface{}]interface{}
	err = grpcClient.GetResource(ctx, deviceID, hubTest.TestResourceLightInstanceHref("1"), &got)
	require.NoError(t, err)

	require.Equal(t, map[interface{}]interface{}{
		"state": true,
		"power": uint64(42),
		"name":  "Light",
	}, got)

	// check applied configurations
	appliedConfs, appliedConfResources := getAppliedConfigurations(ctx, t, snippetClient,
		&pb.GetAppliedConfigurationsRequest{
			DeviceIdFilter: []string{deviceID},
			ConfigurationIdFilter: []*pb.IDFilter{
				{
					Id: conf1.GetId(),
					Version: &pb.IDFilter_All{
						All: true,
					},
				},
				{
					Id: conf2.GetId(),
					Version: &pb.IDFilter_All{
						All: true,
					},
				},
			},
		})
	require.Len(t, appliedConfs, 3)
	require.Len(t, appliedConfResources, 4)

	appliedConfByConfID := make(map[string]*pb.AppliedConfiguration)
	for _, appliedConf := range appliedConfs {
		appliedConfByConfID[appliedConf.GetConfigurationId().GetId()] = appliedConf
	}
	require.Equal(t, cond1.GetId(), appliedConfByConfID[conf1.GetId()].GetConditionId().GetId())
	require.Equal(t, cond2.GetId(), appliedConfByConfID[conf2.GetId()].GetConditionId().GetId())

	notExistingConf1, ok := appliedConfResources[notExistingConf1ID]
	require.True(t, ok)
	require.Equal(t, notExistingResourceHref, notExistingConf1.GetHref())
	require.Equal(t, pb.AppliedConfiguration_Resource_TIMEOUT, notExistingConf1.GetStatus())
	require.Equal(t, commands.Status_ERROR, notExistingConf1.GetResourceUpdated().GetStatus())

	lightConf1ID := conf1.GetId() + "." + hubTest.TestResourceLightInstanceHref("1")
	lightConf1, ok := appliedConfResources[lightConf1ID]
	require.True(t, ok)
	require.Equal(t, hubTest.TestResourceLightInstanceHref("1"), lightConf1.GetHref())
	require.Equal(t, pb.AppliedConfiguration_Resource_DONE, lightConf1.GetStatus())
	require.Equal(t, commands.Status_OK, lightConf1.GetResourceUpdated().GetStatus())
	lightConf2ID := conf2.GetId() + "." + hubTest.TestResourceLightInstanceHref("1")
	lightConf2, ok := appliedConfResources[lightConf2ID]
	require.True(t, ok)
	require.Equal(t, hubTest.TestResourceLightInstanceHref("1"), lightConf2.GetHref())
	require.Equal(t, pb.AppliedConfiguration_Resource_DONE, lightConf2.GetStatus())
	require.Equal(t, commands.Status_OK, lightConf2.GetResourceUpdated().GetStatus())

	conConf3ID := conf3.GetId() + "." + configuration.ResourceURI
	conConf3, ok := appliedConfResources[conConf3ID]
	require.True(t, ok)
	require.Equal(t, configuration.ResourceURI, conConf3.GetHref())
	require.Equal(t, pb.AppliedConfiguration_Resource_DONE, conConf3.GetStatus())
	require.Equal(t, commands.Status_ERROR, conConf3.GetResourceUpdated().GetStatus())

	// restore state
	err = grpcClient.UpdateResource(ctx, deviceID, hubTest.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"state": false,
		"power": uint64(0),
	}, nil)
	require.NoError(t, err)
}

func TestServiceCleanUp(t *testing.T) {
	deviceID := hubTest.MustFindDeviceByName(hubTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	needsShutdown := true
	snippetCfg := test.MakeConfig(t)
	_, shutdownSnippetService := test.New(t, snippetCfg)
	defer func() {
		if needsShutdown {
			needsShutdown = false
			shutdownSnippetService()
		}
	}()

	snippetClientConn, err := grpc.NewClient(config.SNIPPET_SERVICE_HOST, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
		RootCAs: hubTest.GetRootCertificatePool(t),
	})))
	require.NoError(t, err)
	defer func() {
		_ = snippetClientConn.Close()
	}()
	snippetClient := pb.NewSnippetServiceClient(snippetClientConn)

	ctx = pkgGrpc.CtxWithToken(ctx, oauthTest.GetDefaultAccessToken(t))

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		err = grpcClient.Close()
		require.NoError(t, err)
	}()
	_, shutdownDevSim := hubTest.OnboardDevSim(ctx, t, grpcClient.GrpcGatewayClient(), deviceID, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, hubTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	notExistingResourceHref := "/not/existing"
	// configuration
	// -> /not/existing -> { value: 42 }
	conf, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update",
		Owner: oauthService.DeviceUserID,
		Resources: []*pb.Configuration_Resource{
			{
				Href: notExistingResourceHref,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"value": 42,
					}),
				},
				TimeToLive: int64(5 * time.Minute), // will not timeout during test
			},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf.GetId())

	// invoke configuration with long TimeToLive
	resp, err := snippetClient.InvokeConfiguration(ctx, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf.GetId(),
		DeviceId:        deviceID,
	})
	require.NoError(t, err)

	// stop service
	shutdownSnippetService()
	needsShutdown = false

	// check that all configurations are either in timeout or done state
	s, cleanUpStore := test.NewStore(t)
	defer cleanUpStore()

	appliedConfs := make(map[string]*pb.AppliedConfiguration)
	err = s.GetAppliedConfigurations(ctx, oauthService.DeviceUserID, &pb.GetAppliedConfigurationsRequest{
		IdFilter: []string{resp.GetAppliedConfigurationId()},
	}, func(appliedConf *store.AppliedConfiguration) error {
		appliedConfs[appliedConf.GetId()] = appliedConf.GetAppliedConfiguration().Clone()
		return nil
	})
	require.NoError(t, err)
	require.Len(t, appliedConfs, 1)
	appliedConf, ok := appliedConfs[resp.GetAppliedConfigurationId()]
	require.True(t, ok)
	for _, r := range appliedConf.GetResources() {
		status := r.GetStatus()
		require.True(t, pb.AppliedConfiguration_Resource_TIMEOUT == status || pb.AppliedConfiguration_Resource_DONE == status)
	}
}
