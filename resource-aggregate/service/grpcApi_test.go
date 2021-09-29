package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	"github.com/plgd-dev/cloud/pkg/strings"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/publisher"
	natsTest "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventbus/nats/test"
	mongodb "github.com/plgd-dev/cloud/resource-aggregate/cqrs/eventstore/mongodb"
	"github.com/plgd-dev/cloud/resource-aggregate/cqrs/utils"
	"github.com/plgd-dev/cloud/resource-aggregate/service"
	raTest "github.com/plgd-dev/cloud/resource-aggregate/test"
	"github.com/plgd-dev/cloud/test/config"
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)

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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	cfg := raTest.MakeConfig(t)
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

	requestHandler := service.NewRequestHandler(cfg, eventstore, publisher, mockGetOwnerDevices)

	pubReq := testMakePublishResourceRequest(deviceID, []string{href})
	_, err = requestHandler.PublishResourceLinks(ctx, pubReq)
	assert.NoError(t, err)

	for _, tt := range test {
		tfunc := func(t *testing.T) {
			ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
				"sub": tt.args.userID,
			}))
			response, err := requestHandler.UnpublishResourceLinks(ctx, tt.args.request)
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

	ctx := kitNetGrpc.CtxWithIncomingToken(context.Background(), config.CreateJwtToken(t, jwt.MapClaims{
		"sub": user0,
	}))
	config := raTest.MakeConfig(t)
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
			args: args{request: testMakeUpdateResourceRequest(deviceID, resID, "oic.if.baseline", "456", time.Hour)},
			want: &commands.UpdateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
					CorrelationId: "456",
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.UpdateResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
	_, err = requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(deviceID, resID, 0))
	require.NoError(t, err)
	_, err = requestHandler.UpdateResource(ctx, testMakeUpdateResourceRequest(deviceID, resID, "", correlationID, time.Hour))
	require.NoError(t, err)

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
			args: args{request: testMakeRetrieveResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.RetrieveResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
	for _, tt := range test {
		tfunc := func(t *testing.T) {
			if tt.args.request.GetResourceId().GetDeviceId() != "" && tt.args.request.GetResourceId().GetHref() != "" {
				_, err := requestHandler.NotifyResourceChanged(ctx, testMakeNotifyResourceChangedRequest(tt.args.request.GetResourceId().GetDeviceId(), tt.args.request.GetResourceId().GetHref(), 0))
				require.NoError(t, err)
			}
			response, err := requestHandler.RetrieveResource(ctx, tt.args.request)
			if tt.wantError {
				assert.Error(t, err)
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
			args: args{request: testMakeDeleteResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.DeleteResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
			name:      "error-not-found-correlationID",
			args:      args{request: testMakeConfirmResourceDeleteRequest(deviceID, resID, correlationID)},
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
			args: args{request: testMakeCreateResourceRequest(deviceID, resID, correlationID, time.Hour)},
			want: &commands.CreateResourceResponse{
				AuditContext: &commands.AuditContext{
					UserId:        user0,
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
	logger, err := log.NewLogger(config.Log)
	require.NoError(t, err)
	eventstore, err := mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	err = eventstore.Clear(ctx)
	require.NoError(t, err)
	err = eventstore.Close(ctx)
	assert.NoError(t, err)
	eventstore, err = mongodb.New(ctx, config.Clients.Eventstore.Connection.MongoDB, logger, mongodb.WithUnmarshaler(utils.Unmarshal), mongodb.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		err := eventstore.Close(ctx)
		assert.NoError(t, err)
	}()
	naClient, publisher, err := natsTest.NewClientAndPublisher(config.Clients.Eventbus.NATS, logger, publisher.WithMarshaler(utils.Marshal))
	require.NoError(t, err)
	defer func() {
		publisher.Close()
		naClient.Close()
	}()

	requestHandler := service.NewRequestHandler(config, eventstore, publisher, mockGetOwnerDevices)
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
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, response)
		}
		t.Run(tt.name, tfunc)
	}
}

func mockGetOwnerDevices(ctx context.Context, owner string, deviceIDs []string) ([]string, error) {
	ownedDevices, code, err := testListDevicesOfUserFunc(ctx, "0", owner)
	if err != nil {
		return nil, status.Errorf(code, "%v", err)
	}
	getAllDevices := len(deviceIDs) == 0
	if getAllDevices {
		return ownedDevices, nil
	}
	return strings.Intersection(ownedDevices, deviceIDs), nil
}
