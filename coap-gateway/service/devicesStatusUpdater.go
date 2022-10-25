package service

import (
	"context"
	"sync"
	"time"

	"github.com/plgd-dev/hub/v2/pkg/log"
	kitNetGrpc "github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/hub/v2/pkg/time"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
)

type deviceExpires struct {
	expires time.Time
	client  *session
}

type devicesStatusUpdater struct {
	ctx    context.Context
	cfg    DeviceStatusExpirationConfig
	logger log.Logger

	mutex   sync.Mutex
	devices map[string]*deviceExpires
}

func newDevicesStatusUpdater(ctx context.Context, cfg DeviceStatusExpirationConfig, logger log.Logger) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx:     ctx,
		cfg:     cfg,
		devices: make(map[string]*deviceExpires),
		logger:  logger,
	}
	if cfg.Enabled {
		go u.run()
	}
	return &u
}

func (u *devicesStatusUpdater) Add(ctx context.Context, c *session, isNewDevice bool) error {
	now := time.Now()
	connectedAt := time.Time{}
	if isNewDevice {
		connectedAt = now
	}
	expires, err := u.updateOnlineStatus(ctx, c, now.Add(u.cfg.ExpiresIn), connectedAt)
	if err != nil {
		return err
	}
	if !u.cfg.Enabled {
		return nil
	}
	d := deviceExpires{
		client:  c,
		expires: expires,
	}
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.devices[c.RemoteAddr().String()] = &d
	return nil
}

func (u *devicesStatusUpdater) Remove(c *session) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	delete(u.devices, c.RemoteAddr().String())
}

func (u *devicesStatusUpdater) updateOnlineStatus(ctx context.Context, client *session, validUntil time.Time, connectedAt time.Time) (time.Time, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return time.Time{}, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	if !u.cfg.Enabled || authCtx.Expire.UnixNano() < validUntil.UnixNano() {
		validUntil = authCtx.Expire
	}
	_, err = client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: authCtx.GetDeviceID(),
		Update: &commands.UpdateDeviceMetadataRequest_Connection{
			Connection: &commands.Connection{
				Status:           commands.Connection_ONLINE,
				OnlineValidUntil: pkgTime.UnixNano(validUntil),
				Id:               client.RemoteAddr().String(),
				ConnectedAt:      pkgTime.UnixNano(connectedAt),
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.RemoteAddr().String(),
		},
	})

	return validUntil, err
}

func (u *devicesStatusUpdater) getDevicesToUpdate(now time.Time) []*deviceExpires {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	res := make([]*deviceExpires, 0, len(u.devices))
	for key, d := range u.devices {
		select {
		case <-d.client.Context().Done():
			delete(u.devices, key)
		default:
			if d.expires.UnixNano() < now.Add(u.cfg.ExpiresIn/2).UnixNano() {
				res = append(res, d)
			}
		}
	}
	return res
}

func (u *devicesStatusUpdater) run() {
	t := time.NewTicker(u.cfg.ExpiresIn / 10)
	for {
		select {
		case <-u.ctx.Done():
			return
		case now := <-t.C:
			for _, d := range u.getDevicesToUpdate(now) {
				expires, err := u.updateOnlineStatus(d.client.Context(), d.client, time.Now().Add(u.cfg.ExpiresIn), time.Time{})
				if err != nil {
					u.logger.Errorf("cannot update device(%v) status to online: %w", getDeviceID(d.client), err)
				} else {
					d.expires = expires
				}
			}
			u.logger.Debugf("update devices statuses to online takes: %v", time.Since(now))
		}
	}
}
