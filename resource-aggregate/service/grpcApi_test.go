package service

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/kelseyhightower/envconfig"
	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func TestRequestHandler_PublishResource(t *testing.T) {
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	type args struct {
		request *pb.PublishResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.PublishResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{
				request: testMakePublishResourceRequest(deviceId, resId),
			},
			want: &pb.PublishResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:   user0,
					DeviceId: deviceId,
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest(deviceId, resId),
			},
			want: &pb.PublishResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:   user0,
					DeviceId: deviceId,
				},
			},
		},
		{
			args: args{
				request: &pb.PublishResourceRequest{},
			},
			name:      "invalid",
			wantError: true,
		},
	}
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.PublishResource(ctx, tt.args.request)
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"

	type args struct {
		request *pb.UnpublishResourceRequest
		userID  string
	}
	test := []struct {
		name      string
		args      args
		want      *pb.UnpublishResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceId, resId),
				userID:  user0,
			},
			want: &pb.UnpublishResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:   user0,
					DeviceId: deviceId,
				},
			},
		},
		{
			name: "duplicit",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceId, resId),
				userID:  user0,
			},
			wantError: true,
		},
		{
			name: "unauthorized",
			args: args{
				request: testMakeUnpublishResourceRequest(deviceId, resId),
				userID:  testUnauthorizedUser,
			},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &pb.UnpublishResourceRequest{},
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

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(kitNetGrpc.CtxWithIncomingUserID(ctx, user0), pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.UnpublishResource(kitNetGrpc.CtxWithIncomingUserID(ctx, tt.args.userID), tt.args.request)
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"

	type args struct {
		request *pb.NotifyResourceChangedRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.NotifyResourceChangedResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeNotifyResourceChangedRequest(deviceId, resId, 2)},
			want: &pb.NotifyResourceChangedResponse{
				AuditContext: &pb.AuditContext{
					UserId:   user0,
					DeviceId: deviceId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.NotifyResourceChangedRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)

	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.UpdateResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.UpdateResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeUpdateResourceRequest(deviceId, resId, "", correlationId)},
			want: &pb.UpdateResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "valid",
			args: args{request: testMakeUpdateResourceRequest(deviceId, resId, "oic.if.baseline", correlationId)},
			want: &pb.UpdateResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.UpdateResourceRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.ConfirmResourceUpdateRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.ConfirmResourceUpdateResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceUpdateRequest(deviceId, resId, correlationId)},
			want: &pb.ConfirmResourceUpdateResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.ConfirmResourceUpdateRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.RetrieveResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.RetrieveResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeRetrieveResourceRequest(deviceId, resId, correlationId)},
			want: &pb.RetrieveResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.RetrieveResourceRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.ConfirmResourceRetrieveRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.ConfirmResourceRetrieveResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceRetrieveRequest(deviceId, resId, correlationId)},
			want: &pb.ConfirmResourceRetrieveResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.ConfirmResourceRetrieveRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.DeleteResourceRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.DeleteResourceResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeDeleteResourceRequest(deviceId, resId, correlationId)},
			want: &pb.DeleteResourceResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.DeleteResourceRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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
	deviceId := "dev0"
	resId := "res0"
	user0 := "user0"
	correlationId := "123"

	type args struct {
		request *pb.ConfirmResourceDeleteRequest
	}
	test := []struct {
		name      string
		args      args
		want      *pb.ConfirmResourceDeleteResponse
		wantError bool
	}{
		{
			name: "valid",
			args: args{request: testMakeConfirmResourceDeleteRequest(deviceId, resId, correlationId)},
			want: &pb.ConfirmResourceDeleteResponse{
				AuditContext: &pb.AuditContext{
					UserId:        user0,
					DeviceId:      deviceId,
					CorrelationId: correlationId,
				},
			},
		},
		{
			name: "invalid",
			args: args{
				request: &pb.ConfirmResourceDeleteRequest{},
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	var config Config
	err = envconfig.Process("", &config)
	assert.NoError(t, err)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(config, eventstore, publisher, &mockAuthorizationServiceClient{})

	pubReq := testMakePublishResourceRequest(deviceId, resId)
	_, err = requestHandler.PublishResource(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
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

type mockAuthorizationServiceClient struct {
	pbAS.AuthorizationServiceClient
}

type mockGetUserDevicesClientClient struct {
	resourceLink []*pbAS.UserDevice
	i            int
	grpc.ClientStream
}

func (r *mockGetUserDevicesClientClient) Recv() (*pbAS.UserDevice, error) {
	if r.i >= len(r.resourceLink) {
		return nil, io.EOF
	}
	res := r.resourceLink[r.i]
	r.i++
	return res, nil
}

func (c *mockGetUserDevicesClientClient) CloseSend() error {
	return nil
}

func (c mockAuthorizationServiceClient) GetUserDevices(ctx context.Context, in *pbAS.GetUserDevicesRequest, opts ...grpc.CallOption) (pbAS.AuthorizationService_GetUserDevicesClient, error) {
	if len(in.UserIdsFilter) == 0 {
		return nil, fmt.Errorf("UserIdsFilter is empty")
	}

	userID := in.UserIdsFilter[0]

	deviceIds, code, err := testListDevicesOfUserFunc(ctx, "0", userID)
	if err != nil {
		return nil, status.Errorf(code, "%v", err)
	}
	userDevices := make([]*pbAS.UserDevice, 0, 16)
	for _, d := range deviceIds {
		userDevices = append(userDevices, &pbAS.UserDevice{DeviceId: d, UserId: userID})
	}
	return &mockGetUserDevicesClientClient{
		resourceLink: userDevices,
	}, nil
}
