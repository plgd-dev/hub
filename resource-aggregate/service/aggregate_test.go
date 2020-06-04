package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-ocf/go-coap/v2/message"
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
		userID  string
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
				request: testMakePublishResourceRequest("dev0", "res0"),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest("dev0", "res0"),
				userID:  "user0",
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
			deviceIds, _, err := testListDevicesOfUserFunc(ctx, "a0", tt.args.userID)
			ag, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, tt.args.userID), tt.args.request.ResourceId, deviceIds, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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

func testHandlePublishResource(t *testing.T, ctx context.Context, publisher *nats.Publisher, eventstore *mongodb.EventStore, deviceID, resourceID, userID string, expStatusCode codes.Code, hasErr bool) {
	pc := testMakePublishResourceRequest(deviceID, resourceID)

	deviceIds, _, err := testListDevicesOfUserFunc(ctx, "a0", userID)
	assert.NoError(t, err)

	ag, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, userID), resourceID, deviceIds, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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
			assert.Equal(t, userID, resp.AuditContext.UserId)
			assert.Equal(t, deviceID, resp.AuditContext.DeviceId)
		}
		err = publishEvents(ctx, publisher, deviceID, resourceID, events)
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

	deviceID := "dupDeviceId"
	resourceID := "dupResourceId"
	userID := "dupResourceId"

	ag, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, userID), resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	pc1 := testMakePublishResourceRequest(deviceID, resourceID)

	resp1, events, err := ag.PublishResource(ctx, pc1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	ag2, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, userID), resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	pc2 := testMakePublishResourceRequest(deviceID, resourceID)
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

	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	pc := testMakeUnpublishResourceRequest(deviceID, resourceID)

	ag, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, userID), resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	resp, events, err := ag.UnpublishResource(ctx, pc)
	assert.NoError(t, err)
	assert.Equal(t, userID, resp.AuditContext.UserId)
	assert.Equal(t, deviceID, resp.AuditContext.DeviceId)

	err = publishEvents(ctx, publisher, deviceID, resourceID, events)
	assert.NoError(t, err)

	resp, events, err = ag.UnpublishResource(ctx, pc)
	require.Error(t, err)
	s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, s.Code())
	assert.Empty(t, events)
}

func testGetResourceId(deviceID, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+href).String()
}

func testMakePublishResourceRequest(deviceID, resourceID string) *pb.PublishResourceRequest {
	href := "/oic/p"
	r := pb.PublishResourceRequest{
		ResourceId:           resourceID,
		Resource:             testNewResource(href, deviceID, resourceID),
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		TimeToLive:           1,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeUnpublishResourceRequest(deviceID, resourceID string) *pb.UnpublishResourceRequest {
	r := pb.UnpublishResourceRequest{
		ResourceId:           resourceID,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeNotifyResourceChangedRequest(deviceID, resourceID string, seqNum uint64) *pb.NotifyResourceChangedRequest {

	r := pb.NotifyResourceChangedRequest{
		ResourceId:           resourceID,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
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

func testMakeUpdateResourceRequest(deviceID, resourceID, resourceInterface, correlationId string) *pb.UpdateResourceRequest {
	r := pb.UpdateResourceRequest{
		ResourceId:           resourceID,
		ResourceInterface:    resourceInterface,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
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

func testMakeRetrieveResourceRequest(deviceID, resourceID string, correlationId string) *pb.RetrieveResourceRequest {
	r := pb.RetrieveResourceRequest{
		ResourceId:           resourceID,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceUpdateRequest(deviceID, resourceID, correlationId string) *pb.ConfirmResourceUpdateRequest {
	r := pb.ConfirmResourceUpdateRequest{
		ResourceId:           resourceID,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
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

func testMakeConfirmResourceRetrieveRequest(deviceID, resourceID, correlationId string) *pb.ConfirmResourceRetrieveRequest {
	r := pb.ConfirmResourceRetrieveRequest{
		ResourceId:           resourceID,
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
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

func testNewAuthorizationContext(deviceID string) *pb.AuthorizationContext {
	ac := pb.AuthorizationContext{
		DeviceId: deviceID,
	}
	return &ac
}

func testNewResource(href string, deviceID string, resourceID string) *pb.Resource {
	return &pb.Resource{
		Id:                    resourceID,
		Href:                  href,
		ResourceTypes:         []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:            []string{"oic.if.baseline"},
		DeviceId:              deviceID,
		InstanceId:            1,
		Anchor:                "ocf://" + deviceID + "/oic/p",
		Policies:              &pb.Policies{1},
		Title:                 "device",
		SupportedContentTypes: []string{message.TextPlain.String()},
	}
}

func Test_aggregate_HandleNotifyContentChanged(t *testing.T) {
	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

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
					ResourceId:           resourceID,
					AuthorizationContext: testNewAuthorizationContext(deviceID),
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
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 3),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - duplicit",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 2),
			},
			wantResp:       true,
			wantEvents:     false,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - new content",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 5),
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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(kitNetGrpc.CtxWithUserID(ctx, userID), resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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
	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

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
					ResourceId:           resourceID,
					AuthorizationContext: testNewAuthorizationContext(deviceID),
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
				testMakeUpdateResourceRequest(deviceID, resourceID, "", "123"),
			},
			wantResp:       true,
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid with resource interface",
			args: args{
				testMakeUpdateResourceRequest(deviceID, resourceID, "oic.if.baseline", "123"),
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), userID)

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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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
	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

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
					ResourceId:           resourceID,
					AuthorizationContext: testNewAuthorizationContext(deviceID),
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
				testMakeConfirmResourceUpdateRequest(deviceID, resourceID, "123"),
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), userID)

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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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
	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

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
					ResourceId:           resourceID,
					AuthorizationContext: testNewAuthorizationContext(deviceID),
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
				testMakeRetrieveResourceRequest(deviceID, resourceID, "123"),
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), userID)

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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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
	deviceID := "dev0"
	resourceID := "res0"
	userID := "user0"

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
					ResourceId:           resourceID,
					AuthorizationContext: testNewAuthorizationContext(deviceID),
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
				testMakeConfirmResourceRetrieveRequest(deviceID, resourceID, "123"),
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
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), userID)

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

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(ctx, resourceID, []string{deviceID}, 10, eventstore, cqrs.NewDefaultRetryFunc(1))
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

func testListDevicesOfUserFunc(ctx context.Context, correlationId, userID string) ([]string, codes.Code, error) {
	if userID == testUnauthorizedUser {
		return nil, codes.Unauthenticated, fmt.Errorf("unauthorized access")
	}
	deviceIds := []string{"dev0", "dupDeviceId"}
	return deviceIds, codes.OK, nil
}
