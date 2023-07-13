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
	ctx     context.Context
	logger  log.Logger
	cfg     DeviceStatusExpirationConfig
	private struct { // guarded by mutex
		mutex   sync.Mutex
		devices map[string]*deviceExpires
	}
}

func newDevicesStatusUpdater(ctx context.Context, cfg DeviceStatusExpirationConfig, logger log.Logger) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx: ctx,
		cfg: cfg,
		private: struct {
			mutex   sync.Mutex
			devices map[string]*deviceExpires
		}{devices: make(map[string]*deviceExpires)},
		logger: logger,
	}
	if cfg.Enabled {
		go u.run()
	}
	return &u
}

func (u *devicesStatusUpdater) Add(ctx context.Context, c *session, isNewDevice bool) (*commands.UpdateDeviceMetadataResponse, error) {
	now := time.Now()
	connectedAt := time.Time{}
	if isNewDevice {
		connectedAt = now
	}
	resp, expires, err := u.updateOnlineStatus(ctx, c, now.Add(u.cfg.ExpiresIn), connectedAt)
	if err != nil {
		return nil, err
	}
	if !u.cfg.Enabled {
		return resp, nil
	}
	d := deviceExpires{
		client:  c,
		expires: expires,
	}
	u.private.mutex.Lock()
	defer u.private.mutex.Unlock()
	u.private.devices[c.RemoteAddr().String()] = &d
	return resp, nil
}

func (u *devicesStatusUpdater) Remove(c *session) {
	u.private.mutex.Lock()
	defer u.private.mutex.Unlock()
	delete(u.private.devices, c.RemoteAddr().String())
}

func (u *devicesStatusUpdater) updateOnlineStatus(ctx context.Context, client *session, validUntil time.Time, connectedAt time.Time) (*commands.UpdateDeviceMetadataResponse, time.Time, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return nil, time.Time{}, err
	}
	ctx = kitNetGrpc.CtxWithToken(ctx, authCtx.GetAccessToken())
	// When authCtx.Expire is zero, the token will never expire
	if !u.cfg.Enabled || (!authCtx.Expire.IsZero() && authCtx.Expire.UnixNano() < validUntil.UnixNano()) {
		validUntil = authCtx.Expire
	}
	resp, err := client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: authCtx.GetDeviceID(),
		Update: &commands.UpdateDeviceMetadataRequest_Connection{
			Connection: &commands.Connection{
				Status:           commands.Connection_ONLINE,
				OnlineValidUntil: pkgTime.UnixNano(validUntil),
				ConnectedAt:      pkgTime.UnixNano(connectedAt),
				Protocol:         client.GetApplicationProtocol(),
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.RemoteAddr().String(),
		},
	})

	return resp, validUntil, err
}

func (u *devicesStatusUpdater) getDevicesToUpdate(now time.Time) []*deviceExpires {
	u.private.mutex.Lock()
	defer u.private.mutex.Unlock()
	res := make([]*deviceExpires, 0, len(u.private.devices))
	for key, d := range u.private.devices {
		select {
		case <-d.client.Context().Done():
			delete(u.private.devices, key)
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
				_, expires, err := u.updateOnlineStatus(d.client.Context(), d.client, time.Now().Add(u.cfg.ExpiresIn), time.Time{})
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
