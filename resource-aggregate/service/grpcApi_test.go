package service

import (
	"context"
	"github.com/plgd-dev/kit/config"
	"github.com/plgd-dev/kit/log"
	"testing"

	pbAS "github.com/plgd-dev/cloud/authorization/pb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
	"github.com/plgd-dev/kit/security/certManager/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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

	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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

	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
	var cfg Config
	err := config.Load(&cfg)
	assert.NoError(t, err)

	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	log.Set(logger)
	log.Info(cfg.String())

	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), user0)

	natsCertManager, err := client.New(cfg.Clients.Nats.TLSConfig, logger)
	require.NoError(t, err)
	publisher, err := nats.NewPublisher(cfg.Clients.Nats, nats.WithTLS(natsCertManager.GetTLSConfig()))
	assert.NoError(t, err)

	dbCertManager, err := client.New(cfg.Database.MongoDB.TLSConfig, logger)
	require.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(cfg.Database.MongoDB, nil, mongodb.WithTLS(dbCertManager.GetTLSConfig()))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	requestHandler := NewRequestHandler(cfg.Service, eventstore, publisher, mockGetUserDevices)

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
