package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/cloud/pkg/log"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgTime "github.com/plgd-dev/cloud/pkg/time"
	"github.com/plgd-dev/cloud/resource-aggregate/commands"
)

type deviceExpires struct {
	expires time.Time
	client  *Client
}

type devicesStatusUpdater struct {
	ctx context.Context
	cfg DeviceStatusExpirationConfig

	mutex   sync.Mutex
	devices map[string]*deviceExpires
}

func NewDevicesStatusUpdater(ctx context.Context, cfg DeviceStatusExpirationConfig) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx:     ctx,
		cfg:     cfg,
		devices: make(map[string]*deviceExpires),
	}
	if cfg.Enabled {
		go u.run()
	}
	return &u
}

func (u *devicesStatusUpdater) Add(c *Client) error {
	expires, err := u.updateOnlineStatus(c, time.Now().Add(u.cfg.ExpiresIn))
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
	u.devices[c.remoteAddrString()] = &d
	return nil
}

func (u *devicesStatusUpdater) Remove(c *Client) {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	delete(u.devices, c.remoteAddrString())
}

func (u *devicesStatusUpdater) updateOnlineStatus(client *Client, validUntil time.Time) (time.Time, error) {
	authCtx, err := client.GetAuthorizationContext()
	if err != nil {
		return time.Time{}, err
	}
	serviceToken, err := client.server.oauthMgr.GetToken(client.Context())
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot get service token: %w", err)
	}
	ctx := kitNetGrpc.CtxWithOwner(kitNetGrpc.CtxWithToken(client.Context(), serviceToken.AccessToken), authCtx.GetUserID())
	if !u.cfg.Enabled || authCtx.Expire.UnixNano() < validUntil.UnixNano() {
		validUntil = authCtx.Expire
	}
	_, err = client.server.raClient.UpdateDeviceMetadata(ctx, &commands.UpdateDeviceMetadataRequest{
		DeviceId: authCtx.GetDeviceID(),
		Update: &commands.UpdateDeviceMetadataRequest_Status{
			Status: &commands.ConnectionStatus{
				Value:      commands.ConnectionStatus_ONLINE,
				ValidUntil: pkgTime.UnixNano(validUntil),
			},
		},
		CommandMetadata: &commands.CommandMetadata{
			Sequence:     client.coapConn.Sequence(),
			ConnectionId: client.remoteAddrString(),
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
				expires, err := u.updateOnlineStatus(d.client, time.Now().Add(u.cfg.ExpiresIn))
				if err != nil {
					log.Errorf("cannot update device(%v) status to online: %v", getDeviceID(d.client), err)
				} else {
					d.expires = expires
				}
			}
			log.Debugf("update devices statuses to online takes: %v", time.Since(now))
		}
	}
}
