package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/security/certManager"

	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventbus/nats"
	"github.com/go-ocf/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/go-ocf/cloud/resource-aggregate/pb"
	cqrs "github.com/go-ocf/cqrs"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testUnauthorizedUser = "testUnauthorizedUser"

func TestAggregateHandle_PublishResource(t *testing.T) {
	type args struct {
		request *pb.PublishResourceRequest
	}
	test := []struct {
		name    string
		args    args
		want    codes.Code
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				request: testMakePublishResourceRequest("dev0", "res0", "user0"),
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest("dev0", "res0", "user0"),
			},
			want:    codes.OK,
			wantErr: false,
		},
	}

	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)
	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			deviceIds, _, err := testListDevicesOfUserFunc(ctx, "a0", tt.args.request.AuthorizationContext.UserId)
			ag, err := NewAggregate(ctx, tt.args.request.ResourceId, deviceIds, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
			assert.NoError(t, err)
			_, events, err := ag.PublishResource(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
			} else {
				require.NoError(t, err)
				err = publishEvents(ctx, publisher, tt.args.request.AuthorizationContext.DeviceId, tt.args.request.ResourceId, events)
				assert.NoError(t, err)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testHandlePublishResource(t *testing.T, ctx context.Context, publisher *nats.Publisher, eventstore *mongodb.EventStore, deviceId, resourceId, userId string, expStatusCode codes.Code, hasErr bool) {
	pc := testMakePublishResourceRequest(deviceId, resourceId, userId)

	deviceIds, _, err := testListDevicesOfUserFunc(ctx, "a0", userId)
	assert.NoError(t, err)

	ag, err := NewAggregate(ctx, resourceId, deviceIds, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	resp, events, err := ag.PublishResource(ctx, pc)
	if hasErr {
		require.Error(t, err)
		s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
		require.True(t, ok)
		assert.Equal(t, expStatusCode, s.Code())
	} else {
		require.NoError(t, err)
		if err == nil && !assert.NotNil(t, resp.AuditContext) {
			assert.Equal(t, userId, resp.AuditContext.UserId)
			assert.Equal(t, deviceId, resp.AuditContext.DeviceId)
		}
		err = publishEvents(ctx, publisher, deviceId, resourceId, events)
		assert.NoError(t, err)
	}
}

func TestAggregateDuplicitPublishResource(t *testing.T) {
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "token")

	var mgoCfg mongodb.Config
	err = envconfig.Process("", &mgoCfg)
	assert.NoError(t, err)

	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	defer pool.Release()

	eventstore, err := mongodb.NewEventStore(mgoCfg, pool.Submit, mongodb.WithTLS(tlsConfig))

	deviceId := "dupDeviceId"
	resourceId := "dupResourceId"
	userId := "dupResourceId"

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	pc1 := testMakePublishResourceRequest(deviceId, resourceId, userId)

	resp1, events, err := ag.PublishResource(ctx, pc1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	ag2, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	pc2 := testMakePublishResourceRequest(deviceId, resourceId, userId)
	resp2, events, err := ag2.PublishResource(ctx, pc2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	assert.Equal(t, resp1.InstanceId, resp2.InstanceId)
}

func TestAggregateHandleUnpublishResource(t *testing.T) {
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))

	assert.NoError(t, err)
	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	defer pool.Release()

	eventstore, err := mongodb.NewEventStore(mgoCfg, pool.Submit, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	pc := testMakeUnpublishResourceRequest(deviceId, resourceId, userId)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	resp, events, err := ag.UnpublishResource(ctx, pc)
	assert.NoError(t, err)
	assert.Equal(t, userId, resp.AuditContext.UserId)
	assert.Equal(t, deviceId, resp.AuditContext.DeviceId)

	err = publishEvents(ctx, publisher, deviceId, resourceId, events)
	assert.NoError(t, err)

	resp, events, err = ag.UnpublishResource(ctx, pc)
	require.Error(t, err)
	s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, s.Code())
	assert.Empty(t, events)
}

func testGetResourceId(deviceId, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceId+href).String()
}

func testMakePublishResourceRequest(deviceId, resourceId, userId string) *pb.PublishResourceRequest {
	href := "/oic/p"
	r := pb.PublishResourceRequest{
		ResourceId:           resourceId,
		Resource:             testNewResource(href, deviceId, resourceId),
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		TimeToLive:           1,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeUnpublishResourceRequest(deviceId, resourceId string, userId string) *pb.UnpublishResourceRequest {
	r := pb.UnpublishResourceRequest{
		ResourceId:           resourceId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeNotifyResourceChangedRequest(deviceId, resourceId string, userId string, seqNum uint64) *pb.NotifyResourceChangedRequest {

	r := pb.NotifyResourceChangedRequest{
		ResourceId:           resourceId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		Content: &pb.Content{
			Data: []byte("hello world"),
		},
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: "test",
			Sequence:     seqNum,
		},
	}
	return &r
}

func testMakeUpdateResourceRequest(deviceId, resourceId, resourceInterface, userId, correlationId string) *pb.UpdateResourceRequest {
	r := pb.UpdateResourceRequest{
		ResourceId:           resourceId,
		ResourceInterface:    resourceInterface,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		Content: &pb.Content{
			Data: []byte("hello world"),
		},
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeRetrieveResourceRequest(deviceId, resourceId string, userId string, correlationId string) *pb.RetrieveResourceRequest {
	r := pb.RetrieveResourceRequest{
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceUpdateRequest(deviceId, resourceId, userId, correlationId string) *pb.ConfirmResourceUpdateRequest {
	r := pb.ConfirmResourceUpdateRequest{
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		Content: &pb.Content{
			Data: []byte("hello world"),
		},
		Status: pb.Status_OK,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceRetrieveRequest(deviceId, resourceId, userId, correlationId string) *pb.ConfirmResourceRetrieveRequest {
	r := pb.ConfirmResourceRetrieveRequest{
		ResourceId:           resourceId,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
		Content: &pb.Content{
			Data: []byte("hello world"),
		},
		Status: pb.Status_OK,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testNewAuthorizationContext(deviceId string, userId string) *pb.AuthorizationContext {
	ac := pb.AuthorizationContext{
		DeviceId: deviceId,
		UserId:   userId,
	}
	return &ac
}

func testNewResource(href string, deviceId string, resourceId string) *pb.Resource {
	return &pb.Resource{
		Id:                    resourceId,
		Href:                  href,
		ResourceTypes:         []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:            []string{"oic.if.baseline"},
		DeviceId:              deviceId,
		InstanceId:            1,
		Anchor:                "ocf://" + deviceId + "/oic/p",
		Policies:              &pb.Policies{1},
		Title:                 "device",
		SupportedContentTypes: []string{coap.TextPlain.String()},
	}
}

func Test_aggregate_HandleNotifyContentChanged(t *testing.T) {
	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	type args struct {
		req *pb.NotifyResourceChangedRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResp       bool
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.NotifyResourceChangedRequest{
					ResourceId:           resourceId,
					AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
				},
			},
			wantResp:       false,
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceId, resourceId, userId, 3),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - duplicit",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceId, resourceId, userId, 2),
			},
			wantResp:       true,
			wantEvents:     false,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - new content",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceId, resourceId, userId, 5),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotEvents, err := ag.NotifyResourceChanged(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			if tt.wantResp {
				assert.NotEmpty(t, gotResp)
			}
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleUpdateResourceContent(t *testing.T) {
	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	type args struct {
		req *pb.UpdateResourceRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResp       bool
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.UpdateResourceRequest{
					ResourceId:           resourceId,
					AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
				},
			},
			wantResp:       false,
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeUpdateResourceRequest(deviceId, resourceId, "", userId, "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid with resource interface",
			args: args{
				testMakeUpdateResourceRequest(deviceId, resourceId, "oic.if.baseline", userId, "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotEvents, err := ag.UpdateResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			if tt.wantResp {
				assert.NotEmpty(t, gotResp)
			}
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleConfirmResourceUpdate(t *testing.T) {
	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	type args struct {
		req *pb.ConfirmResourceUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResp       bool
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.ConfirmResourceUpdateRequest{
					ResourceId:           resourceId,
					AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
				},
			},
			wantResp:       false,
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceUpdateRequest(deviceId, resourceId, userId, "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotEvents, err := ag.ConfirmResourceUpdate(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			if tt.wantResp {
				assert.NotEmpty(t, gotResp)
			}
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleRetrieveResource(t *testing.T) {
	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	type args struct {
		req *pb.RetrieveResourceRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResp       bool
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.RetrieveResourceRequest{
					ResourceId:           resourceId,
					AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
				},
			},
			wantResp:       false,
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeRetrieveResourceRequest(deviceId, resourceId, userId, "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	eventstore, err := mongodb.NewEventStore(mgoCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotEvents, err := ag.RetrieveResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			if tt.wantResp {
				assert.NotEmpty(t, gotResp)
			}
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleNotifyResourceContentResourceProcessed(t *testing.T) {
	deviceId := "dev0"
	resourceId := "res0"
	userId := "user0"

	type args struct {
		req *pb.ConfirmResourceRetrieveRequest
	}
	tests := []struct {
		name           string
		args           args
		wantResp       bool
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.ConfirmResourceRetrieveRequest{
					ResourceId:           resourceId,
					AuthorizationContext: testNewAuthorizationContext(deviceId, userId),
				},
			},
			wantResp:       false,
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceRetrieveRequest(deviceId, resourceId, userId, "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceId, resourceId, userId, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceId, []string{deviceId}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotEvents, err := ag.ConfirmResourceRetrieve(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			if tt.wantResp {
				assert.NotEmpty(t, gotResp)
			}
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func testListDevicesOfUserFunc(ctx context.Context, correlationId, userId string) ([]string, codes.Code, error) {
	if userId == testUnauthorizedUser {
		return nil, codes.Unauthenticated, fmt.Errorf("unauthorized access")
	}
	deviceIds := []string{"dev0", "dupDeviceId"}
	return deviceIds, codes.OK, nil
}
