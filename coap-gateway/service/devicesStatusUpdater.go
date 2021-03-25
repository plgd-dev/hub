package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	deviceStatus "github.com/plgd-dev/cloud/coap-gateway/schema/device/status"
	kitNetGrpc "github.com/plgd-dev/cloud/pkg/net/grpc"
	pbCQRS "github.com/plgd-dev/cloud/resource-aggregate/pb"
	"github.com/plgd-dev/kit/log"
)

type deviceExpires struct {
	expires time.Time
	client  *Client
}

type devicesStatusUpdater struct {
	ctx                  context.Context
	deviceStatusValidity time.Duration

	mutex   sync.Mutex
	devices map[string]*deviceExpires
}

func NewDevicesStatusUpdater(ctx context.Context, deviceStatusValidity time.Duration) *devicesStatusUpdater {
	u := devicesStatusUpdater{
		ctx:                  ctx,
		deviceStatusValidity: deviceStatusValidity,
		devices:              make(map[string]*deviceExpires),
	}
	go u.run()
	return &u
}

func (u *devicesStatusUpdater) Add(c *Client) error {
	expires, err := u.updateOnlineStatus(c, time.Now().Add(u.deviceStatusValidity))
	if err != nil {
		return err
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
	authCtx := client.loadAuthorizationContext()
	if isExpired(authCtx.Expire) {
		return time.Time{}, fmt.Errorf("token is expired")
	}
	serviceToken, err := client.server.oauthMgr.GetToken(client.Context())
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot get service token: %w", err)
	}
	ctx := kitNetGrpc.CtxWithUserID(kitNetGrpc.CtxWithToken(client.Context(), serviceToken.AccessToken), authCtx.GetUserID())
	if authCtx.Expire.Before(validUntil) {
		validUntil = authCtx.Expire
	}

	return validUntil, deviceStatus.SetOnline(ctx, client.server.raClient, authCtx.GetDeviceId(), validUntil, &pbCQRS.CommandMetadata{
		Sequence:     client.coapConn.Sequence(),
		ConnectionId: client.remoteAddrString(),
	}, authCtx.AuthorizationContext)
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
			if now.Add(u.deviceStatusValidity / 2).After(d.expires) {
				res = append(res, d)
			}
		}
	}
	return res
}

func (u *devicesStatusUpdater) run() {
	t := time.NewTicker(u.deviceStatusValidity / 10)
	for {
		select {
		case <-u.ctx.Done():
			return
		case now := <-t.C:
			for _, d := range u.getDevicesToUpdate(now) {
				expires, err := u.updateOnlineStatus(d.client, time.Now().Add(u.deviceStatusValidity))
				if err != nil {
					log.Errorf("cannot update device(%v) status to online: %v", getDeviceID(d.client), err)
				} else {
					d.expires = expires
				}
			}
		}
	}
}
