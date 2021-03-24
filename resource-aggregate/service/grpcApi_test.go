package service

import (
	"context"
	"testing"

	"github.com/kelseyhightower/envconfig"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_PublishResource(t *testing.T) {
	deviceID := "dev0"
	href := "/res0"
	user0 := "user0"
	type args struct {
		request *commands.PublishResourceLinksRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.PublishResourceLinksResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{
				request: testMakePublishResourceRequest(deviceID, []string{href}),
			},
			want: &commands.PublishResourceLinksResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
				PublishedResources: []*commands.Resource{testNewResource(href, deviceID)},
				DeviceId:           deviceID,
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest(deviceID, []string{href}),
			},
			want: &commands.PublishResourceLinksResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
				PublishedResources: []*commands.Resource{},
				DeviceId:           deviceID,
			},
		},
		{
			name: "invalid href",
			args: args{
				request: testMakePublishResourceRequest(deviceID, []string{"hrefwithoutslash"}),
			},
			wantError: true,
		},
		{
			name: "empty href",
			args: args{
				request: testMakePublishResourceRequest(deviceID, []string{""}),
			},
			wantError: true,
		},
		{
			name: "root href",
			args: args{
				request: testMakePublishResourceRequest(deviceID, []string{"/"}),
			},
			wantError: true,
		},
		{
			name: "empty",
			args: args{
				request: &commands.PublishResourceLinksRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.PublishResourceLinks(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.want != nil {
				assert.Equal(t, tt.want.AuditContext, response.AuditContext)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_UnpublishResource(t *testing.T) {
	deviceID := "dev0"
	href := "/res0"
	user0 := "user0"

	type args struct {
		request *commands.UnpublishResourceLinksRequest
		userID  string
	}
	test := []struct {
		name      string
		args      args
		want      *commands.UnpublishResourceLinksResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceID, []string{href}),
				userID:  user0,
			},
			want: &commands.UnpublishResourceLinksResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
				UnpublishedHrefs: []string{href},
				DeviceId:         deviceID,
			},
		},
		{
			name: "unauthorized",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceID, []string{href}),
				userID:  testUnauthorizedUser,
			},
			wantError: true,
		},
		{
			name: "invalid href",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceID, []string{"hrefwithoutslash"}),
			},
			wantError: true,
		},
		{
			name: "empty href",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceID, []string{""}),
			},
			wantError: true,
		},
		{
			name: "root href",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceID, []string{"/"}),
			},
			wantError: true,
		},
		{
			name: "empty",
			args: args{
				request: &commands.UnpublishResourceLinksRequest{},
			},
			wantError: true,
		},
	}

	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)

	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = requestHandler.PublishResourceLinks(kitNetGrpc.CtxWithIncomingOwner(ctx, user0), pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.UnpublishResourceLinks(kitNetGrpc.CtxWithIncomingOwner(ctx, tt.args.userID), tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_NotifyResourceChanged(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"

	type args struct {
		request *commands.NotifyResourceChangedRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.NotifyResourceChangedResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeNotifyResourceChangedRequest(deviceID, resID, 2)},
			want: &commands.NotifyResourceChangedResponse{
				AuditContext: &commands.AuditContext{
					UserId: user0,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.NotifyResourceChangedRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.NotifyResourceChanged(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_UpdateResourceContent(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.UpdateResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.UpdateResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeUpdateResourceRequest(deviceID, resID, "", correlationID)},
			want: &commands.UpdateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "valid",
			args: args{request: testMakeUpdateResourceRequest(deviceID, resID, "oic.if.baseline", correlationID)},
			want: &commands.UpdateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.UpdateResourceRequest{},
			},
			wantError: true,
		},
	}

	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.UpdateResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_ConfirmResourceUpdate(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.ConfirmResourceUpdateRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.ConfirmResourceUpdateResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceUpdateRequest(deviceID, resID, correlationID)},
			want: &commands.ConfirmResourceUpdateResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceUpdateRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceUpdate(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_RetrieveResource(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.RetrieveResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.RetrieveResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeRetrieveResourceRequest(deviceID, resID, correlationID)},
			want: &commands.RetrieveResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.RetrieveResourceRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.RetrieveResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_ConfirmResourceRetrieve(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.ConfirmResourceRetrieveRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.ConfirmResourceRetrieveResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceRetrieveRequest(deviceID, resID, correlationID)},
			want: &commands.ConfirmResourceRetrieveResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceRetrieveRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceRetrieve(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_DeleteResource(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.DeleteResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.DeleteResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeDeleteResourceRequest(deviceID, resID, correlationID)},
			want: &commands.DeleteResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.DeleteResourceRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.DeleteResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_ConfirmResourceDelete(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.ConfirmResourceDeleteRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.ConfirmResourceDeleteResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceDeleteRequest(deviceID, resID, correlationID)},
			want: &commands.ConfirmResourceDeleteResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceDeleteRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceDelete(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_CreateResource(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.CreateResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.CreateResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeCreateResourceRequest(deviceID, resID, correlationID)},
			want: &commands.CreateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.CreateResourceRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.CreateResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandler_ConfirmResourceCreate(t *testing.T) {
	deviceID := "dev0"
	resID := "/res0"
	user0 := "user0"
	correlationID := "123"

	type args struct {
		request *commands.ConfirmResourceCreateRequest
	}
	test := []struct {
		name      string
		args      args
		want      *commands.ConfirmResourceCreateResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceCreateRequest(deviceID, resID, correlationID)},
			want: &commands.ConfirmResourceCreateResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceCreateRequest{},
			},
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingOwner(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(ctx, jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, mockGetUserDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceCreate(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

type mockAuthorizationServiceClient struct {
	pbAS.AuthorizationServiceClient
}

func mockGetUserDevices(ctx context.Context, userID, deviceID string) (bool, error) {
	deviceIds, code, err := testListDevicesOfUserFunc(ctx, "0", userID)
	if err != nil {
		return false, status.Errorf(code, "%v", err)
	}
	for _, id := range deviceIds {
		if id == deviceID {
			return true, nil
		}
	}
	return false, nil
}
