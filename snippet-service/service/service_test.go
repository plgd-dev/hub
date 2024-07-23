package service_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	deviceClient "github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	bridgeDevice "github.com/plgd-dev/device/v2/cmd/bridge-device/device"
	deviceCoap "github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/go-coap/v3/message"
	grpcgwTest "github.com/plgd-dev/hub/v2/grpc-gateway/test"
	"github.com/plgd-dev/hub/v2/pkg/config/database"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/mongodb"
	pkgGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/server"
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
	"github.com/plgd-dev/hub/v2/test/device/bridge"
	oauthService "github.com/plgd-dev/hub/v2/test/oauth-server/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/sdk"
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

	tearDown := hubTestService.SetUpServices(ctx, t, hubTestService.SetUpServicesOAuth|hubTestService.SetUpServicesMachine2MachineOAuth)
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
					Storage: storeConfig.Config{
						Config: database.Config[*storeMongo.Config, *storeCqlDB.Config]{
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
					Storage: storeConfig.Config{
						Config: database.Config[*storeMongo.Config, *storeCqlDB.Config]{
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
					Storage: storeConfig.Config{
						Config: database.Config[*storeMongo.Config, *storeCqlDB.Config]{
							Use:   database.CqlDB,
							CqlDB: &storeCqlDB.Config{},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid expired updates checker config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients.Storage.CleanUpExpiredUpdates = "invalid"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid resource subscriber config",
			cfg: func() service.Config {
				cfg := test.MakeConfig(t)
				cfg.Clients.EventBus.NATS = natsClient.ConfigSubscriber{}
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
				cfg.APIs.GRPC.Authorization = server.AuthorizationConfig{}
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
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	snippetCfg := test.MakeConfig(t)
	const interval = time.Second
	snippetCfg.Clients.Storage.CleanUpExpiredUpdates = "*/1 * * * * *"
	snippetCfg.Clients.Storage.ExtendCronParserBySeconds = true
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

	defer func() {
		// restore state
		err = grpcClient.UpdateResource(ctx, deviceID, hubTest.TestResourceLightInstanceHref("1"), map[string]interface{}{
			"state": false,
			"power": uint64(0),
		}, nil)
		require.NoError(t, err)
	}()

	// -> wait for /conf1 to be applied -> for /not/existing resource this should start-up the timeout timer
	notExistingConf1ID := conf1.GetId() + "." + notExistingResourceHref
	var appliedConf1Status pb.AppliedConfiguration_Resource_Status
	test.WaitForAppliedConfigurations(ctx, t, snippetClient, &pb.GetAppliedConfigurationsRequest{
		DeviceIdFilter: []string{deviceID},
		ConfigurationIdFilter: []*pb.IDFilter{
			{
				Id: conf1.GetId(),
				Version: &pb.IDFilter_All{
					All: true,
				},
			},
		},
	}, map[string][]pb.AppliedConfiguration_Resource_Status{
		hubTest.TestResourceLightInstanceHref("1"): {pb.AppliedConfiguration_Resource_DONE},
		notExistingResourceHref:                    {pb.AppliedConfiguration_Resource_TIMEOUT},
	})
	require.NotEqual(t, pb.AppliedConfiguration_Resource_QUEUED, appliedConf1Status)

	if appliedConf1Status != pb.AppliedConfiguration_Resource_TIMEOUT &&
		appliedConf1Status != pb.AppliedConfiguration_Resource_DONE {
		// -> wait enough time to timeout pending commands
		time.Sleep(2 * interval)
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
	appliedConfs, appliedConfResources := test.GetAppliedConfigurations(ctx, t, snippetClient,
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
}

func getBridgeDeviceResources(ctx context.Context, t *testing.T, bd *bridge.Device, numResources int) (map[string]map[string]interface{}, func()) {
	sdkClient, err := sdk.NewClient(bd.GetSDKClientOptions()...)
	require.NoError(t, err)
	defer func() {
		errC := sdkClient.Close(context.Background())
		require.NoError(t, errC)
	}()

	deviceID, err := sdkClient.OwnDevice(ctx, bd.GetID(), deviceClient.WithOTM(deviceClient.OTMType_JustWorks))
	require.NoError(t, err)
	bd.SetID(deviceID)

	// get resource from device via SDK
	bdResources := make(map[string]map[string]interface{}, numResources)
	for i := range numResources {
		var bdResource map[string]interface{}
		err = sdkClient.GetResource(ctx, bd.GetID(), bridgeDevice.GetTestResourceHref(i), &bdResource)
		require.NoError(t, err)
		bdResources[bridgeDevice.GetTestResourceHref(i)] = bdResource
	}

	return bdResources, func() {
		for href, content := range bdResources {
			err = sdkClient.UpdateResource(ctx, bd.GetID(), href, content, nil)
			require.NoError(t, err)
		}
	}
}

func TestServiceWithBridgedDevice(t *testing.T) {
	bdConfig, err := hubTest.GetBridgeDeviceConfig()
	require.NoError(t, err)

	if bdConfig.NumGeneratedBridgedDevices == 0 || bdConfig.NumResourcesPerDevice == 0 {
		t.Skip("no bridge device with resources running")
	}
	bdName := hubTest.TestBridgeDeviceInstanceName("0")
	bdID := hubTest.MustFindDeviceByName(bdName, func(d *core.Device) deviceCoap.OptionFunc {
		return deviceCoap.WithQuery("di=" + d.DeviceID())
	})
	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT)
	defer cancel()

	tearDown := hubTestService.SetUp(ctx, t)
	defer tearDown()

	snippetCfg := test.MakeConfig(t)
	ss, shutdownSnippetService := test.New(t, snippetCfg)
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

	// configuration
	// -> /test/%i -> { name: "new name" }
	notExistingResourceHref := "/not/existing"
	conf, err := snippetClient.CreateConfiguration(ctx, &pb.Configuration{
		Name:  "update name",
		Owner: oauthService.DeviceUserID,
		Resources: func() []*pb.Configuration_Resource {
			var resources []*pb.Configuration_Resource
			for i := 0; i < bdConfig.NumResourcesPerDevice; i++ {
				resources = append(resources, &pb.Configuration_Resource{
					Href: bridgeDevice.GetTestResourceHref(i),
					Content: &commands.Content{
						ContentType: message.AppOcfCbor.String(),
						Data: hubTest.EncodeToCbor(t, map[string]interface{}{
							"name": "new name",
						}),
					},
				})
			}
			resources = append(resources, &pb.Configuration_Resource{
				Href: notExistingResourceHref,
				Content: &commands.Content{
					ContentType: message.AppOcfCbor.String(),
					Data: hubTest.EncodeToCbor(t, map[string]interface{}{
						"value": 42,
					}),
				},
			})
			return resources
		}(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, conf.GetId())

	// condition for configuration
	_, err = snippetClient.CreateCondition(ctx, &pb.Condition{
		Owner:              oauthService.DeviceUserID,
		Enabled:            true,
		ConfigurationId:    conf.GetId(),
		DeviceIdFilter:     []string{bdID},
		ResourceTypeFilter: []string{bridgeDevice.TestResourceType},
		ApiAccessToken:     token,
	})
	require.NoError(t, err)

	grpcClient := grpcgwTest.NewTestClient(t)
	defer func() {
		err = grpcClient.Close()
		require.NoError(t, err)
	}()

	bd := bridge.NewDevice(bdID, bdName, bdConfig.NumResourcesPerDevice, true)
	originalResources, restoreOriginalResources := getBridgeDeviceResources(ctx, t, bd, bdConfig.NumResourcesPerDevice)
	defer restoreOriginalResources()
	require.NotEmpty(t, originalResources)

	shutdownBd := hubTest.OnboardDevice(ctx, t, grpcClient.GrpcGatewayClient(), bd, config.ACTIVE_COAP_SCHEME+"://"+config.COAP_GW_HOST, bd.GetDefaultResources())
	defer shutdownBd()

	appliedConfResources := test.WaitForAppliedConfigurations(ctx, t, snippetClient, &pb.GetAppliedConfigurationsRequest{
		ConfigurationIdFilter: []*pb.IDFilter{
			{
				Id:      conf.GetId(),
				Version: &pb.IDFilter_All{All: true},
			},
		},
	}, func() map[string][]pb.AppliedConfiguration_Resource_Status {
		statusFilter := make(map[string][]pb.AppliedConfiguration_Resource_Status)
		for i := range bdConfig.NumResourcesPerDevice {
			statusFilter[bridgeDevice.GetTestResourceHref(i)] = []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_DONE}
		}
		return statusFilter
	}())
	require.Len(t, appliedConfResources, bdConfig.NumResourcesPerDevice+1)

	// force invoke configuration
	_, err = snippetClient.InvokeConfiguration(ctx, &pb.InvokeConfigurationRequest{
		ConfigurationId: conf.GetId(),
		DeviceId:        bdID,
		Force:           true,
	})
	require.NoError(t, err)
	appliedConfResources = test.WaitForAppliedConfigurations(ctx, t, snippetClient, &pb.GetAppliedConfigurationsRequest{
		ConfigurationIdFilter: []*pb.IDFilter{
			{
				Id:      conf.GetId(),
				Version: &pb.IDFilter_All{All: true},
			},
		},
	}, func() map[string][]pb.AppliedConfiguration_Resource_Status {
		statusFilter := make(map[string][]pb.AppliedConfiguration_Resource_Status)
		for i := range bdConfig.NumResourcesPerDevice {
			statusFilter[bridgeDevice.GetTestResourceHref(i)] = []pb.AppliedConfiguration_Resource_Status{pb.AppliedConfiguration_Resource_DONE}
		}
		return statusFilter
	}())
	require.Len(t, appliedConfResources, bdConfig.NumResourcesPerDevice+1)

	// cancel pending update of not existing resource
	err = ss.CancelPendingResourceUpdates(ctx)
	require.NoError(t, err)
}
