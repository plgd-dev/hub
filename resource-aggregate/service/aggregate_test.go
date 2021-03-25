package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/kit/security/certManager"

	"github.com/gofrs/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/panjf2000/ants/v2"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	cqrsAggregate "github.com/plgd-dev/cloud/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/pb"
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
				request: testMakePublishResourceRequest("dev0", "/oic/p"),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest("dev0", "/oic/p"),
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
	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	assert.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	assert.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := NewAggregate(tt.args.request.ResourceId, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			events, err := ag.PublishResource(kitNetGrpc.CtxWithIncomingUserID(ctx, tt.args.userID), tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
			} else {
				require.NoError(t, err)
				err = publishEvents(ctx, publisher, tt.args.request.ResourceId.GetDeviceId(), ag.resourceID, events)
				assert.NoError(t, err)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testHandlePublishResource(t *testing.T, ctx context.Context, publisher *nats.Publisher, eventstore EventStore, deviceID, href, userID string, expStatusCode codes.Code, hasErr bool) {
	pc := testMakePublishResourceRequest(deviceID, href)

	ag, err := NewAggregate(pc.GetResourceId(), 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	events, err := ag.PublishResource(ctx, pc)
	if hasErr {
		require.Error(t, err)
		s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
		require.True(t, ok)
		assert.Equal(t, expStatusCode, s.Code())
	} else {
		require.NoError(t, err)
		err = publishEvents(ctx, publisher, deviceID, ag.resourceID, events)
		assert.NoError(t, err)
	}
}

func TestAggregateDuplicitPublishResource(t *testing.T) {
	deviceID := "dupDeviceId"
	resourceID := "dupResourceId"
	userID := "dupResourceId"
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "token"), userID)

	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	defer pool.Release()

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc1 := testMakePublishResourceRequest(deviceID, resourceID)

	events, err := ag.PublishResource(ctx, pc1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	ag2, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc2 := testMakePublishResourceRequest(deviceID, resourceID)
	events, err = ag2.PublishResource(ctx, pc2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
}

func TestAggregateHandleUnpublishResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"
	var cmconfig certManager.Config
	err := envconfig.Process("DIAL", &cmconfig)
	assert.NoError(t, err)
	dialCertManager, err := certManager.NewCertManager(cmconfig)
	require.NoError(t, err)
	tlsConfig := dialCertManager.GetClientTLSConfig()
	ctx := kitNetGrpc.CtxWithIncomingUserID(kitNetGrpc.CtxWithIncomingToken(context.Background(), "b"), userID)

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))

	assert.NoError(t, err)
	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	defer pool.Release()

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	pc := testMakeUnpublishResourceRequest(deviceID, resourceID)

	ag, err := NewAggregate(pc.GetResourceId(), 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	events, err := ag.UnpublishResource(ctx, pc)
	assert.NoError(t, err)

	err = publishEvents(ctx, publisher, deviceID, resourceID, events)
	assert.NoError(t, err)

	events, err = ag.UnpublishResource(ctx, pc)
	require.Error(t, err)
	s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, s.Code())
	assert.Empty(t, events)
}

func testGetResourceId(deviceID, href string) string {
	return uuid.NewV5(uuid.NamespaceURL, deviceID+href).String()
}

func testMakePublishResourceRequest(deviceID, href string) *pb.PublishResourceRequest {
	r := pb.PublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Resource:             testNewResource(href, deviceID, utils.MakeResourceId(deviceID, href)),
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		TimeToLive:           1,
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeUnpublishResourceRequest(deviceID, href string) *pb.UnpublishResourceRequest {
	r := pb.UnpublishResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeNotifyResourceChangedRequest(deviceID, href string, seqNum uint64) *pb.NotifyResourceChangedRequest {

	r := pb.NotifyResourceChangedRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
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

func testMakeUpdateResourceRequest(deviceID, href, resourceInterface, correlationId string) *pb.UpdateResourceRequest {
	r := pb.UpdateResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
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

func testMakeRetrieveResourceRequest(deviceID, href string, correlationId string) *pb.RetrieveResourceRequest {
	r := pb.RetrieveResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeDeleteResourceRequest(deviceID, href string, correlationId string) *pb.DeleteResourceRequest {
	r := pb.DeleteResourceRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId:        correlationId,
		AuthorizationContext: testNewAuthorizationContext(deviceID),
		CommandMetadata: &pb.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewV4()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceUpdateRequest(deviceID, href, correlationId string) *pb.ConfirmResourceUpdateRequest {
	r := pb.ConfirmResourceUpdateRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
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

func testMakeConfirmResourceRetrieveRequest(deviceID, href, correlationId string) *pb.ConfirmResourceRetrieveRequest {
	r := pb.ConfirmResourceRetrieveRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
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

func testMakeConfirmResourceDeleteRequest(deviceID, href, correlationId string) *pb.ConfirmResourceDeleteRequest {
	r := pb.ConfirmResourceDeleteRequest{
		ResourceId: &pb.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
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
		Id:            resourceID,
		Href:          href,
		ResourceTypes: []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:    []string{"oic.if.baseline"},
		DeviceId:      deviceID,
		Anchor:        "ocf://" + deviceID + "/oic/p",
		Policies: &pb.Policies{
			BitFlags: 1,
		},
		Title:                 "device",
		SupportedContentTypes: []string{message.TextPlain.String()},
	}
}

func Test_aggregate_HandleNotifyContentChanged(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.NotifyResourceChangedRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.NotifyResourceChangedRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 3),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - duplicit",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 2),
			},
			wantEvents:     false,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid - new content",
			args: args{
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 5),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)

	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.NotifyResourceChanged(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleUpdateResourceContent(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.UpdateResourceRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.UpdateResourceRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeUpdateResourceRequest(deviceID, resourceID, "", "123"),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid with resource interface",
			args: args{
				testMakeUpdateResourceRequest(deviceID, resourceID, "oic.if.baseline", "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)

	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()
	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.UpdateResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleConfirmResourceUpdate(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.ConfirmResourceUpdateRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.ConfirmResourceUpdateRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceUpdateRequest(deviceID, resourceID, "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.ConfirmResourceUpdate(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleRetrieveResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.RetrieveResourceRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.RetrieveResourceRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeRetrieveResourceRequest(deviceID, resourceID, "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.RetrieveResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleNotifyResourceContentResourceProcessed(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.ConfirmResourceRetrieveRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.ConfirmResourceRetrieveRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceRetrieveRequest(deviceID, resourceID, "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.ConfirmResourceRetrieve(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
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

func Test_aggregate_HandleDeleteResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.DeleteResourceRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.DeleteResourceRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeDeleteResourceRequest(deviceID, resourceID, "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.DeleteResource(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}

func Test_aggregate_HandleConfirmResourceDelete(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *pb.ConfirmResourceDeleteRequest
	}
	tests := []struct {
		name           string
		args           args
		wantEvents     bool
		wantStatusCode codes.Code
		wantErr        bool
	}{
		{
			name: "invalid",
			args: args{
				&pb.ConfirmResourceDeleteRequest{
					ResourceId:           &pb.ResourceId{},
					AuthorizationContext: testNewAuthorizationContext(deviceID),
					Status:               pb.Status_OK,
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceDeleteRequest(deviceID, resourceID, "123"),
			},
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

	var natsCfg nats.Config
	err = envconfig.Process("", &natsCfg)
	assert.NoError(t, err)
	publisher, err := nats.NewPublisher(natsCfg, nats.WithTLS(tlsConfig))
	assert.NoError(t, err)

	var jsmCfg mongodb.Config
	err = envconfig.Process("", &jsmCfg)
	assert.NoError(t, err)
	eventstore, err := mongodb.NewEventStore(jsmCfg, nil, mongodb.WithTLS(tlsConfig))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Clear(ctx)
		assert.NoError(t, err)
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, deviceID, resourceID, userID, codes.OK, false)

	ag, err := NewAggregate(&pb.ResourceId{DeviceId: deviceID, Href: resourceID}, 10, eventstore, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.ConfirmResourceDelete(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)

			if tt.wantEvents {
				assert.NotEmpty(t, gotEvents)
			}
		})
	}
}
