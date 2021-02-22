package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	deviceStatus "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
	kitNetGrpc "github.com/plgd-dev/kit/net/grpc"
)

type deviceExpires struct {
	expires time.Time
	client  *Client
}

type devicesStatusUpdater struct {
	ctx context.Context
	cfg DeviceStatusValidityConfig

	mutex   sync.Mutex
	devices map[string]*deviceExpires
}

func NewDevicesStatusUpdater(ctx context.Context, cfg DeviceStatusValidityConfig) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx:     ctx,
		cfg:     cfg,
		devices: make(map[string]*deviceExpires),
	}
	if cfg.Enable {
		go u.run()
	}
	return &u
}

func (u *devicesStatusUpdater) Add(c *Client) error {
	expires, err := u.updateOnlineStatus(c, time.Now().Add(u.cfg.Validity))
	if err != nil {
		return err
	}
	if !u.cfg.Enable {
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
	authCtx, err := client.loadAuthorizationContext()
	if err != nil {
		return time.Time{}, err
	}
	serviceToken, err := client.server.oauthMgr.GetToken(client.Context())
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot get service token: %w", err)
	}
	ctx := kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(client.Context(), serviceToken.AccessToken), authCtx.GetUserID())
	if !u.cfg.Enable || authCtx.Expire.Before(validUntil) {
		validUntil = authCtx.Expire
	}

	return validUntil, deviceStatus.SetOnline(ctx, client.server.raClient, authCtx.GetDeviceID(), validUntil, &pbCQRS.CommandMetadata{
		Sequence:     client.coapConn.Sequence(),
		ConnectionId: client.remoteAddrString(),
	}, authCtx.GetPbData())
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
			if now.Add(u.cfg.Validity / 2).After(d.expires) {
				res = append(res, d)
			}
		}
	}
	return res
}

func (u *devicesStatusUpdater) run() {
	t := time.NewTicker(u.cfg.Validity / 10)
	for {
		select {
		case <-u.ctx.Done():
			return
		case now := <-t.C:
			for _, d := range u.getDevicesToUpdate(now) {
				expires, err := u.updateOnlineStatus(d.client, time.Now().Add(u.cfg.Validity))
				if err != nil {
					log.Errorf("cannot update device(%v) status to online: %v", getDeviceID(d.client), err)
				} else {
					d.expires = expires
				}
			}
			log.Debugf("update devices statuses to online takes: %v", time.Now().Sub(now))
		}
	}
}
