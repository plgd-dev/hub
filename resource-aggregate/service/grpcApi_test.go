package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/hub/v2/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/hub/v2/resource-aggregate/service"
	raTest "github.com/plgd-dev/hub/v2/resource-aggregate/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/status"
)

func TestRequestHandlerPublishResource(t *testing.T) {
	deviceID := dev0
	const href = "/res0"
	const user0 = "user0"
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
					Owner:  user0,
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
					Owner:  user0,
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.PublishResourceLinks(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if tt.want != nil {
				require.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
			}
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerUnpublishResource(t *testing.T) {
	deviceID := dev0
	const href = "/res0"
	const user0 = "user0"

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
					Owner:  user0,
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	cfg := raTest.MakeConfig(t)
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

	serviceHeartbeat := service.NewServiceHeartbeat(cfg, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = requestHandler.PublishResourceLinks(ctx, pubReq)
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			response, err := requestHandler.UnpublishResourceLinks(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerNotifyResourceChanged(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"

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
					Owner:  user0,
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.NotifyResourceChanged(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerUpdateResourceContent(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"

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
			args: args{request: testMakeUpdateResourceRequest(deviceID, resID, "", "123", time.Hour)},
			want: &commands.UpdateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: "123",
					Owner:         user0,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
			},
		},
		{
			name:      "error-duplicit-correlationID",
			args:      args{request: testMakeUpdateResourceRequest(deviceID, resID, "", "123", time.Hour)},
			wantError: true,
		},
		{
			name: "valid with interface",
			args: args{request: testMakeUpdateResourceRequest(deviceID, resID, interfaces.OC_IF_BASELINE, "456", time.Hour)},
			want: &commands.UpdateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: "456",
					Owner:         user0,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.UpdateResource(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want.GetValidUntil() == 0 {
				assert.Equal(t, tt.want.GetValidUntil(), response.GetValidUntil())
			} else {
				assert.Less(t, tt.want.GetValidUntil(), response.GetValidUntil())
			}
			assert.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmResourceUpdate(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
					Owner:         user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name:      "error-not-found-correlationID",
			args:      args{request: testMakeConfirmResourceUpdateRequest(deviceID, resID, correlationID)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceUpdateRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	_, err = requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID, 0))
	require.NoError(t, err)
	_, err = requestHandler.UpdateResource(ctx, testMakeUpdateResourceRequest(deviceID, resID, "", correlationID, time.Hour))
	require.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			response, err := requestHandler.ConfirmResourceUpdate(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerRetrieveResource(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
			args: args{request: testMakeRetrieveResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.RetrieveResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					Owner:         user0,
					CorrelationId: correlationID,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
			},
		},
		{
			name:      "error-duplicit-correlationID",
			args:      args{request: testMakeRetrieveResourceRequest(deviceID, resID, correlationID, time.Hour)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.RetrieveResourceRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()
	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.RetrieveResource(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want.GetValidUntil() == 0 {
				assert.Equal(t, tt.want.GetValidUntil(), response.GetValidUntil())
			} else {
				assert.Less(t, tt.want.GetValidUntil(), response.GetValidUntil())
			}
			assert.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmResourceRetrieve(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
					Owner:         user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name:      "error-not-found-correlationID",
			args:      args{request: testMakeConfirmResourceRetrieveRequest(deviceID, resID, correlationID)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceRetrieveRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	_, err = requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID, 0))
	require.NoError(t, err)
	_, err = requestHandler.RetrieveResource(ctx, testMakeRetrieveResourceRequest(deviceID, resID, correlationID, time.Hour))
	require.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceRetrieve(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerDeleteResource(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
			args: args{request: testMakeDeleteResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.DeleteResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					Owner:         user0,
					CorrelationId: correlationID,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
			},
		},
		{
			name:      "error-duplicit-correlationID",
			args:      args{request: testMakeDeleteResourceRequest(deviceID, resID, correlationID, time.Hour)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.DeleteResourceRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.DeleteResource(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want.GetValidUntil() == 0 {
				assert.Equal(t, tt.want.GetValidUntil(), response.GetValidUntil())
			} else {
				assert.Less(t, tt.want.GetValidUntil(), response.GetValidUntil())
			}
			assert.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmResourceDelete(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
			args: args{request: testMakeConfirmResourceDeleteRequest(deviceID, resID, correlationID, commands.Status_OK)},
			want: &commands.ConfirmResourceDeleteResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					Owner:         user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name:      "error-not-found-correlationID",
			args:      args{request: testMakeConfirmResourceDeleteRequest(deviceID, resID, correlationID, commands.Status_FORBIDDEN)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceDeleteRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	_, err = requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID, 0))
	require.NoError(t, err)
	_, err = requestHandler.DeleteResource(ctx, testMakeDeleteResourceRequest(deviceID, resID, correlationID, time.Hour))
	require.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceDelete(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerCreateResource(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
			args: args{request: testMakeCreateResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.CreateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					Owner:         user0,
					CorrelationId: correlationID,
				},
				ValidUntil: pkgTime.UnixNano(time.Now().Add(time.Hour)),
			},
		},
		{
			name:      "error-duplicit-correlationID",
			args:      args{request: testMakeCreateResourceRequest(deviceID, resID, correlationID, time.Hour)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.CreateResourceRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.CreateResource(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want.GetValidUntil() == 0 {
				assert.Equal(t, tt.want.GetValidUntil(), response.GetValidUntil())
			} else {
				assert.Less(t, tt.want.GetValidUntil(), response.GetValidUntil())
			}
			assert.Equal(t, tt.want.GetAuditContext(), response.GetAuditContext())
		}
		t.Run(tt.name, tfunc)
	}
}

func TestRequestHandlerConfirmResourceCreate(t *testing.T) {
	deviceID := dev0
	const resID = "/res0"
	const user0 = "user0"
	const correlationID = "123"

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
					Owner:         user0,
					CorrelationId: correlationID,
				},
			},
		},
		{
			name:      "not-found",
			args:      args{request: testMakeConfirmResourceCreateRequest(deviceID, resID, correlationID)},
			wantError: true,
		},
		{
			name: "invalid",
			args: args{
				request: &commands.ConfirmResourceCreateRequest{},
			},
			wantError: true,
		},
	}

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger := log.NewLogger(config.Log)
	fileWatcher, err := fsnotify.NewWatcher(logger)
	require.NoError(t, err)
	defer func() {
		errC := fileWatcher.Close()
		require.NoError(t, errC)
	}()
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	require.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, fileWatcher, logger, noop.NewTracerProvider(), mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		errC := eventstore.Close(ctx)
		require.NoError(t, errC)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, fileWatcher, logger, noop.NewTracerProvider(), publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	serviceHeartbeat := service.NewServiceHeartbeat(config, eventstore, publisher, logger)
	defer serviceHeartbeat.Close()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices, serviceHeartbeat, logger)

	_, err = requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID, 0))
	require.NoError(t, err)
	_, err = requestHandler.CreateResource(ctx, testMakeCreateResourceRequest(deviceID, resID, correlationID, time.Hour))
	require.NoError(t, err)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.ConfirmResourceCreate(ctx, tt.args.request)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func mockGetOwnerDevices(_ context.Context, owner string, deviceIDs []string) ([]string, error) {
	ownedDevices, code, err := testListDevicesOfUserFunc(owner)
	if err != nil {
		return nil, status.Errorf(code, "%v", err)
	}
	getAllDevices := len(deviceIDs) == 0
	if getAllDevices {
		return ownedDevices, nil
	}
	return strings.Intersection(ownedDevices, deviceIDs), nil
}
