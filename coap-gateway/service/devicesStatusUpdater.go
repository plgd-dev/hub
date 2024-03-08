package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

type devicesStatusUpdater struct {
	ctx               context.Context
	logger            log.Logger
	serviceInstanceID uuid.UUID
}

func newDevicesStatusUpdater(ctx context.Context, serviceInstanceID uuid.UUID, logger log.Logger) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx:               ctx,
		serviceInstanceID: serviceInstanceID,
		logger:            logger,
	}
	return &u
}

func (u *devicesStatusUpdater) UpdateOnlineStatus(ctx context.Context, c *session) (*commands.UpdateDeviceMetadataResponse, error) {
	return u.updateOnlineStatus(ctx, c, time.Now())
}

func (u *devicesStatusUpdater) updateOnlineStatus(ctx context.Context, client *session, connectedAt time.Time) (*commands.UpdateDeviceMetadataResponse, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	resp, err := client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: authCtx.GetDeviceID(),
		Update: &commands.UpdateDeviceMetadataRequest_Connection{
			Connection: &commands.Connection{
				Status:         commands.Connection_ONLINE,
				ConnectedAt:    pkgTime.UnixNano(connectedAt),
				Protocol:       client.GetApplicationProtocol(),
				ServiceId:      u.serviceInstanceID.String(),
				LocalEndpoints: client.getLocalEndpoints(),
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.RemoteAddr().String(),
		},
	})

	return resp, err
}
