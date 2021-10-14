package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/hub/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/pkg/net/grpc"
	"github.com/plgd-dev/hub/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/resource-aggregate/cqrs/utils"
	raEvents "github.com/plgd-dev/hub/resource-aggregate/events"
	"github.com/plgd-dev/hub/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/resource-aggregate/test"
	"github.com/plgd-dev/hub/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	testUnauthorizedUser = "testUnauthorizedUser"
	testUserDevices      = []string{"dev0", "dev1", "dev2", "dupDeviceId"}
)

func TestAggregateHandle_PublishResourceLinks(t *testing.T) {
	type args struct {
		request *commands.PublishResourceLinksRequest
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
				request: testMakePublishResourceRequest("dev0", []string{"/oic/p"}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "valid multiple",
			args: args{
				request: testMakePublishResourceRequest("dev0", []string{"/oic/p", "/oic/d"}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest("dev0", []string{"/oic/p"}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	logger, err := log.NewLogger(cfg.Log)

	fmt.Printf("%v\n", cfg.String())

	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	assert.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			ctx = kitNetGrpc.CtxWithIncomingToken(ctx, config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			events, err := ag.PublishResourceLinks(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				assert.Equal(t, tt.want, s.Code())
			} else {
				require.NoError(t, err)
				err = service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events)
				assert.NoError(t, err)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testHandlePublishResource(t *testing.T, ctx context.Context, publisher *publisher.Publisher, eventstore service.EventStore, userID, deviceID string, hrefs []string, expStatusCode codes.Code, hasErr bool) {
	pc := testMakePublishResourceRequest(deviceID, hrefs)

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	events, err := ag.PublishResourceLinks(ctx, pc)
	if hasErr {
		require.Error(t, err)
		s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
		require.True(t, ok)
		assert.Equal(t, expStatusCode, s.Code())
	} else {
		require.NoError(t, err)
		err = service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events)
		assert.NoError(t, err)
	}
}

func TestAggregateDuplicitPublishResource(t *testing.T) {
	deviceID := "dupDeviceId"
	resourceID := "/dupResourceId"
	userID := "dupResourceId"

	pool, err := ants.NewPool(16)
	assert.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc1 := testMakePublishResourceRequest(deviceID, []string{resourceID})

	events, err := ag.PublishResourceLinks(ctx, pc1)
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))

	ag2, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc2 := testMakePublishResourceRequest(deviceID, []string{resourceID})
	events, err = ag2.PublishResourceLinks(ctx, pc2)
	require.NoError(t, err)
	assert.Empty(t, events)

	ag3, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc3 := testMakePublishResourceRequest(deviceID, []string{resourceID, resourceID, resourceID})
	events, err = ag3.PublishResourceLinks(ctx, pc3)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestAggregateHandleUnpublishResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, userID, deviceID, []string{resourceID}, codes.OK, false)

	pc := testMakeUnpublishResourceRequest(deviceID, []string{resourceID})

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	assert.NoError(t, err)

	err = service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events)
	assert.NoError(t, err)

	_, err = ag.UnpublishResourceLinks(ctx, pc)
	assert.NoError(t, err)
}

func TestAggregateHandleUnpublishAllResources(t *testing.T) {
	deviceID := "dev0"
	resourceID1 := "/res1"
	resourceID2 := "/res2"
	resourceID3 := "/res3"
	userID := "user0"
	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, userID, deviceID, []string{resourceID1, resourceID2, resourceID3}, codes.OK, false)

	pc := testMakeUnpublishResourceRequest(deviceID, []string{})

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))

	unpublishedResourceLinks := (events[0].(*raEvents.ResourceLinksUnpublished)).Hrefs
	assert.Equal(t, 3, len(unpublishedResourceLinks))
	assert.Contains(t, unpublishedResourceLinks, resourceID1, resourceID2, resourceID3)

	err = service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events)
	assert.NoError(t, err)

	events, err = ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestAggregateHandleUnpublishResourceSubset(t *testing.T) {
	deviceID := "dev0"
	resourceID1 := "/res1"
	resourceID2 := "/res2"
	resourceID3 := "/res3"
	resourceID4 := "/res4"
	userID := "user0"
	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, userID, deviceID, []string{resourceID1, resourceID2, resourceID3, resourceID4}, codes.OK, false)

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), 10, eventstore, service.ResourceLinksFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)
	pc := testMakeUnpublishResourceRequest(deviceID, []string{resourceID1, resourceID3})
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, []string{resourceID1, resourceID3}, (events[0].(*raEvents.ResourceLinksUnpublished)).Hrefs)

	err = service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events)
	assert.NoError(t, err)

	pc = testMakeUnpublishResourceRequest(deviceID, []string{resourceID1, resourceID4, resourceID4})
	events, err = ag.UnpublishResourceLinks(ctx, pc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, []string{resourceID4}, (events[0].(*raEvents.ResourceLinksUnpublished)).Hrefs)
}

func testMakePublishResourceRequest(deviceID string, href []string) *commands.PublishResourceLinksRequest {
	resources := []*commands.Resource{}
	for _, h := range href {
		resources = append(resources, testNewResource(h, deviceID))
	}
	r := commands.PublishResourceLinksRequest{
		Resources: resources,
		DeviceId:  deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeUnpublishResourceRequest(deviceID string, hrefs []string) *commands.UnpublishResourceLinksRequest {
	r := commands.UnpublishResourceLinksRequest{
		Hrefs:    hrefs,
		DeviceId: deviceID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeNotifyResourceChangedRequest(deviceID, href string, seqNum uint64, content ...byte) *commands.NotifyResourceChangedRequest {
	if len(content) == 0 {
		content = []byte("hello world")
	}
	r := commands.NotifyResourceChangedRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		Content: &commands.Content{
			Data: content,
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: "test",
			Sequence:     seqNum,
		},
	}
	return &r
}

func testMakeUpdateResourceRequest(deviceID, href, resourceInterface, correlationID string, timeToLive time.Duration) *commands.UpdateResourceRequest {
	r := commands.UpdateResourceRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		ResourceInterface: resourceInterface,
		CorrelationId:     correlationID,
		TimeToLive:        int64(timeToLive),
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeRetrieveResourceRequest(deviceID, href string, correlationID string, timeToLive time.Duration) *commands.RetrieveResourceRequest {
	r := commands.RetrieveResourceRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		TimeToLive:    int64(timeToLive),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeDeleteResourceRequest(deviceID, href string, correlationID string, timeToLive time.Duration) *commands.DeleteResourceRequest {
	r := commands.DeleteResourceRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		TimeToLive:    int64(timeToLive),
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeCreateResourceRequest(deviceID, href string, correlationID string, timeToLive time.Duration) *commands.CreateResourceRequest {
	r := commands.CreateResourceRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		TimeToLive: int64(timeToLive),
		Content: &commands.Content{
			Data: []byte("create hello world"),
		},
		CorrelationId: correlationID,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceCreateRequest(deviceID, href, correlationID string) *commands.ConfirmResourceCreateRequest {
	r := commands.ConfirmResourceCreateRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		Status: commands.Status_OK,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceUpdateRequest(deviceID, href, correlationID string) *commands.ConfirmResourceUpdateRequest {
	r := commands.ConfirmResourceUpdateRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		Status: commands.Status_OK,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceRetrieveRequest(deviceID, href, correlationID string) *commands.ConfirmResourceRetrieveRequest {
	r := commands.ConfirmResourceRetrieveRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		Status: commands.Status_OK,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testMakeConfirmResourceDeleteRequest(deviceID, href, correlationID string) *commands.ConfirmResourceDeleteRequest {
	r := commands.ConfirmResourceDeleteRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		Status: commands.Status_OK,
		CommandMetadata: &commands.CommandMetadata{
			ConnectionId: uuid.Must(uuid.NewRandom()).String(),
			Sequence:     0,
		},
	}
	return &r
}

func testNewResource(href string, deviceID string) *commands.Resource {
	return &commands.Resource{
		Href:          href,
		DeviceId:      deviceID,
		ResourceTypes: []string{"oic.wk.d", "x.org.iotivity.device"},
		Interfaces:    []string{"oic.if.baseline"},
		Anchor:        "ocf://" + deviceID + "/oic/p",
		Policy: &commands.Policy{
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
		req *commands.NotifyResourceChangedRequest
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
				&commands.NotifyResourceChangedRequest{
					ResourceId: &commands.ResourceId{},
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
				testMakeNotifyResourceChangedRequest(deviceID, resourceID, 5, []byte("new content")...),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(t, ctx, publisher, eventstore, userID, deviceID, []string{resourceID}, codes.OK, false)

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
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
		req   *commands.UpdateResourceRequest
		sleep time.Duration
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
				req: &commands.UpdateResourceRequest{
					ResourceId: &commands.ResourceId{},
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid with validUntil",
			args: args{
				req:   testMakeUpdateResourceRequest(deviceID, resourceID, "", "123", time.Millisecond*250),
				sleep: time.Millisecond * 500,
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid with same correlationID",
			args: args{
				req: testMakeUpdateResourceRequest(deviceID, resourceID, "", "123", time.Minute),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid with resource interface",
			args: args{
				req: testMakeUpdateResourceRequest(deviceID, resourceID, "oic.if.baseline", "456", time.Minute),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "invalid valid until",
			args: args{
				req: testMakeUpdateResourceRequest(deviceID, resourceID, "", "123", -time.Hour),
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}

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
			time.Sleep(tt.args.sleep)
		})
	}
}

func Test_aggregate_HandleConfirmResourceUpdate(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *commands.ConfirmResourceUpdateRequest
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
				&commands.ConfirmResourceUpdateRequest{
					ResourceId: &commands.ResourceId{},
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

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

	_, err = ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resourceID, 0))
	require.NoError(t, err)
	_, err = ag.UpdateResource(ctx, testMakeUpdateResourceRequest(deviceID, resourceID, "", "123", 0))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
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
		req   *commands.RetrieveResourceRequest
		sleep time.Duration
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
				req: &commands.RetrieveResourceRequest{
					ResourceId: &commands.ResourceId{},
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid with validUntil",
			args: args{
				req:   testMakeRetrieveResourceRequest(deviceID, resourceID, "123", time.Millisecond*250),
				sleep: time.Millisecond * 500,
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid",
			args: args{
				req: testMakeRetrieveResourceRequest(deviceID, resourceID, "123", time.Hour),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "invalid valid until",
			args: args{
				req: testMakeRetrieveResourceRequest(deviceID, resourceID, "123", -time.Hour),
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
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
			time.Sleep(tt.args.sleep)
		})
	}
}

func Test_aggregate_HandleNotifyResourceContentResourceProcessed(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *commands.ConfirmResourceRetrieveRequest
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
				&commands.ConfirmResourceRetrieveRequest{
					ResourceId: &commands.ResourceId{},
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

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	_, err = ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resourceID, 0))
	require.NoError(t, err)
	_, err = ag.RetrieveResource(ctx, testMakeRetrieveResourceRequest(deviceID, resourceID, "123", time.Hour))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
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

func testListDevicesOfUserFunc(ctx context.Context, correlationID, userID string) ([]string, codes.Code, error) {
	if userID == testUnauthorizedUser {
		return nil, codes.Unauthenticated, fmt.Errorf("unauthorized access")
	}
	return testUserDevices, codes.OK, nil
}

func Test_aggregate_HandleDeleteResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req   *commands.DeleteResourceRequest
		sleep time.Duration
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
				req: &commands.DeleteResourceRequest{
					ResourceId: &commands.ResourceId{},
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid with validUntil",
			args: args{
				req:   testMakeDeleteResourceRequest(deviceID, resourceID, "123", time.Millisecond*250),
				sleep: time.Millisecond * 500,
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid",
			args: args{
				req:   testMakeDeleteResourceRequest(deviceID, resourceID, "123", time.Hour),
				sleep: time.Millisecond * 500,
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "invalid valid until",
			args: args{
				req: testMakeDeleteResourceRequest(deviceID, resourceID, "123", -time.Hour),
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
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
			time.Sleep(tt.args.sleep)
		})
	}
}

func Test_aggregate_HandleConfirmResourceDelete(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *commands.ConfirmResourceDeleteRequest
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
				&commands.ConfirmResourceDeleteRequest{
					ResourceId: &commands.ResourceId{},
					Status:     commands.Status_OK,
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

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	_, err = ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resourceID, 0))
	require.NoError(t, err)
	_, err = ag.DeleteResource(ctx, testMakeDeleteResourceRequest(deviceID, resourceID, "123", time.Hour))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
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

func Test_aggregate_HandleCreateResource(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req   *commands.CreateResourceRequest
		sleep time.Duration
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
				req: &commands.CreateResourceRequest{
					ResourceId: &commands.ResourceId{},
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid with validUntil",
			args: args{
				req:   testMakeCreateResourceRequest(deviceID, resourceID, "123", time.Millisecond*250),
				sleep: time.Millisecond * 500,
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "valid",
			args: args{
				req: testMakeCreateResourceRequest(deviceID, resourceID, "123", time.Hour),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
		{
			name: "invalid valid until",
			args: args{
				req: testMakeCreateResourceRequest(deviceID, resourceID, "123", -time.Hour),
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			gotEvents, err := ag.CreateResource(ctx, tt.args.req)
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
			time.Sleep(tt.args.sleep)
		})
	}
}

func Test_aggregate_HandleConfirmResourceCreate(t *testing.T) {
	deviceID := "dev0"
	resourceID := "/oic/p"
	userID := "user0"

	type args struct {
		req *commands.ConfirmResourceCreateRequest
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
				&commands.ConfirmResourceCreateRequest{
					ResourceId: &commands.ResourceId{},
					Status:     commands.Status_OK,
				},
			},
			wantEvents:     false,
			wantStatusCode: codes.InvalidArgument,
			wantErr:        true,
		},
		{
			name: "valid",
			args: args{
				testMakeConfirmResourceCreateRequest(deviceID, resourceID, "123"),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger, err := log.NewLogger(cfg.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), 10, eventstore, service.ResourceStateFactoryModel, cqrsAggregate.NewDefaultRetryFunc(1))
	assert.NoError(t, err)

	_, err = ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resourceID, 0))
	require.NoError(t, err)
	_, err = ag.CreateResource(ctx, testMakeCreateResourceRequest(deviceID, resourceID, "123", time.Hour))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.req.GetResourceId().GetDeviceId() != "" && tt.args.req.GetResourceId().GetHref() != "" {
				_, err := ag.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.req.GetResourceId().GetDeviceId(), tt.args.req.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			gotEvents, err := ag.ConfirmResourceCreate(ctx, tt.args.req)
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
