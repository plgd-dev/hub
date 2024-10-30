package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	cqrsAggregate "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/aggregate"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	raEvents "github.com/plgd-dev/hub/v2/resource-aggregate/events"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	hubTest "github.com/plgd-dev/hub/v2/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	testUnauthorizedUser = "testUnauthorizedUser"
	dev0                 = hubTest.GenerateDeviceIDbyIdx(0)
	dev1                 = hubTest.GenerateDeviceIDbyIdx(1)
	dev2                 = hubTest.GenerateDeviceIDbyIdx(2)
	dupDeviceId          = hubTest.GenerateDeviceIDbyIdx(99)
	testUserDevices      = []string{dev0, dev1, dev2, dupDeviceId}
)

func TestAggregateHandlePublishResourceLinks(t *testing.T) {
	type args struct {
		request *commands.PublishResourceLinksRequest
		userID  string
		owner   string
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
				request: testMakePublishResourceRequest(dev0, []string{platform.ResourceURI}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "valid multiple",
			args: args{
				request: testMakePublishResourceRequest(dev0, []string{platform.ResourceURI, device.ResourceURI}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
		{
			name: "duplicit",
			args: args{
				request: testMakePublishResourceRequest(dev0, []string{platform.ResourceURI}),
				userID:  "user0",
			},
			want:    codes.OK,
			wantErr: false,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), "b")
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()

	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	require.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ag, err := service.NewAggregate(commands.NewResourceID(tt.args.request.GetDeviceId(), commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(tt.args.userID, tt.args.owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
			require.NoError(t, err)
			events, err := ag.PublishResourceLinks(ctx, tt.args.request)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				require.Equal(t, tt.want, s.Code())
			} else {
				require.NoError(t, err)
				service.PublishEvents(publisher, tt.args.userID, tt.args.request.GetDeviceId(), ag.ResourceID(), events, logger)
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func testHandlePublishResource(ctx context.Context, t *testing.T, publisher *publisher.Publisher, eventstore *mongodb.EventStore, userID, deviceID, owner, hubID string, hrefs []string) {
	pc := testMakePublishResourceRequest(deviceID, hrefs)

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, hubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	events, err := ag.PublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events, log.Get())
}

func TestAggregateDuplicitPublishResource(t *testing.T) {
	deviceID := dupDeviceId
	const resourceID = "/dupResourceId"
	const userID = "dupResourceId"
	const owner = "owner"

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc1 := testMakePublishResourceRequest(deviceID, []string{resourceID})

	events, err := ag.PublishResourceLinks(ctx, pc1)
	require.NoError(t, err)
	require.Len(t, events, 1)

	ag2, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc2 := testMakePublishResourceRequest(deviceID, []string{resourceID})
	events, err = ag2.PublishResourceLinks(ctx, pc2)
	require.NoError(t, err)
	require.Empty(t, events)

	ag3, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc3 := testMakePublishResourceRequest(deviceID, []string{resourceID, resourceID, resourceID})
	events, err = ag3.PublishResourceLinks(ctx, pc3)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestAggregateHandleUnpublishResource(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": userID,
	}))
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(ctx, t, publisher, eventstore, userID, deviceID, owner, cfg.HubID, []string{resourceID})

	pc := testMakeUnpublishResourceRequest(deviceID, []string{resourceID})

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)

	service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events, logger)

	_, err = ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
}

func TestAggregateHandleUnpublishAllResources(t *testing.T) {
	deviceID := dev1
	const resourceID1 = "/res1"
	const resourceID2 = "/res2"
	const resourceID3 = "/res3"
	const userID = "user1"
	const owner = "owner1"
	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(ctx, t, publisher, eventstore, userID, deviceID, owner, cfg.HubID, []string{resourceID1, resourceID2, resourceID3})

	pc := testMakeUnpublishResourceRequest(deviceID, []string{})

	ag, err := service.NewAggregate(commands.NewResourceID(pc.GetDeviceId(), commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	require.Len(t, events, 1)

	unpublishedResourceLinks := (events[0].(*raEvents.ResourceLinksUnpublished)).GetHrefs()
	require.Len(t, unpublishedResourceLinks, 3)
	require.Contains(t, unpublishedResourceLinks, resourceID1, resourceID2, resourceID3)

	service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events, logger)

	events, err = ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestAggregateHandleUnpublishResourceSubset(t *testing.T) {
	deviceID := dev2
	const resourceID1 = "/res1"
	const resourceID2 = "/res2"
	const resourceID3 = "/res3"
	const resourceID4 = "/res4"
	const userID = "user2"
	const owner = "owner2"
	pool, err := ants.NewPool(16)
	require.NoError(t, err)
	defer pool.Release()

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(ctx, t, publisher, eventstore, userID, deviceID, owner, cfg.HubID, []string{resourceID1, resourceID2, resourceID3, resourceID4})

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, commands.ResourceLinksHref), eventstore, service.NewResourceLinksFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)
	pc := testMakeUnpublishResourceRequest(deviceID, []string{resourceID1, resourceID3})
	events, err := ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, []string{resourceID1, resourceID3}, (events[0].(*raEvents.ResourceLinksUnpublished)).GetHrefs())

	service.PublishEvents(publisher, userID, deviceID, ag.ResourceID(), events, logger)

	pc = testMakeUnpublishResourceRequest(deviceID, []string{resourceID1, resourceID4, resourceID4})
	events, err = ag.UnpublishResourceLinks(ctx, pc)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, []string{resourceID4}, (events[0].(*raEvents.ResourceLinksUnpublished)).GetHrefs())
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

func testMakeConfirmResourceDeleteRequest(deviceID, href, correlationID string, status commands.Status) *commands.ConfirmResourceDeleteRequest {
	r := commands.ConfirmResourceDeleteRequest{
		ResourceId: &commands.ResourceId{
			DeviceId: deviceID,
			Href:     href,
		},
		CorrelationId: correlationID,
		Content: &commands.Content{
			Data: []byte("hello world"),
		},
		Status: status,
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
		ResourceTypes: []string{device.ResourceType, "x.org.iotivity.device"},
		Interfaces:    []string{interfaces.OC_IF_BASELINE},
		Anchor:        "ocf://" + deviceID + platform.ResourceURI,
		Policy: &commands.Policy{
			BitFlags: 1,
		},
		Title:                 "device",
		SupportedContentTypes: []string{message.TextPlain.String()},
	}
}

func TestAggregateHandleNotifyContentChanged(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(cfg.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	testHandlePublishResource(ctx, t, publisher, eventstore, userID, deviceID, owner, cfg.HubID, []string{resourceID})

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEvents, err := ag.NotifyResourceChanged(ctx, tt.args.req)
			if tt.wantErr {
				require.Error(t, err)
				s, ok := status.FromError(kitNetGrpc.ForwardFromError(codes.Unknown, err))
				require.True(t, ok)
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
		})
	}
}

func TestAggregateHandleUpdateResourceContent(t *testing.T) {
	deviceID := dev1
	const resourceID = platform.ResourceURI
	const userID = "user1"
	const owner = "owner1"

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
				req: testMakeUpdateResourceRequest(deviceID, resourceID, interfaces.OC_IF_BASELINE, "456", time.Minute),
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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
			time.Sleep(tt.args.sleep)
		})
	}
}

func TestAggregateHandleConfirmResourceUpdate(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
		})
	}
}

func TestAggregateHandleRetrieveResource(t *testing.T) {
	deviceID := dev1
	const resourceID = platform.ResourceURI
	const userID = "user1"
	const owner = "owner1"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
			time.Sleep(tt.args.sleep)
		})
	}
}

func TestAggregateHandleNotifyResourceContentResourceProcessed(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
		})
	}
}

func testListDevicesOfUserFunc(userID string) ([]string, codes.Code, error) {
	if userID == testUnauthorizedUser {
		return nil, codes.Unauthenticated, errors.New("unauthorized access")
	}
	return testUserDevices, codes.OK, nil
}

func TestAggregateHandleDeleteResource(t *testing.T) {
	deviceID := dev1
	const resourceID = platform.ResourceURI
	const userID = "user1"
	const owner = "owner1"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
			time.Sleep(tt.args.sleep)
		})
	}
}

func TestAggregateHandleConfirmResourceDelete(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

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
				testMakeConfirmResourceDeleteRequest(deviceID, resourceID, "123", commands.Status_OK),
			},
			wantEvents:     true,
			wantStatusCode: codes.OK,
			wantErr:        false,
		},
	}

	cfg := raTest.MakeConfig(t)
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)

			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
		})
	}
}

func TestAggregateHandleCreateResource(t *testing.T) {
	deviceID := dev1
	const resourceID = platform.ResourceURI
	const userID = "user1"
	const owner = "owner1"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)
			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
			time.Sleep(tt.args.sleep)
		})
	}
}

func TestAggregateHandleConfirmResourceCreate(t *testing.T) {
	deviceID := dev0
	const resourceID = platform.ResourceURI
	const userID = "user0"
	const owner = "owner0"

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
	ctx := context.Background()
	logger := log.NewLogger(cfg.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, cfg.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()

	ag, err := service.NewAggregate(commands.NewResourceID(deviceID, resourceID), eventstore, service.NewResourceStateFactoryModel(userID, owner, cfg.HubID), cqrsAggregate.NewDefaultRetryFunc(1))
	require.NoError(t, err)

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
				require.Equal(t, tt.wantStatusCode, s.Code())
				return
			}
			require.NoError(t, err)

			if tt.wantEvents {
				require.NotEmpty(t, gotEvents)
			}
		})
	}
}
