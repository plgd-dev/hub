package test

import (
	"context"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
)

// Default test observer handler
//
// It implements ServiceHandler interface by just logging the called method and
// returning default response and no error (if required).
type DefaultObserverHandler struct {
	deviceID            string
	accessTokenLifetime time.Duration
}

func MakeDefaultObserverHandler(accessTokenLifetime time.Duration) DefaultObserverHandler {
	return DefaultObserverHandler{
		accessTokenLifetime: accessTokenLifetime,
	}
}

func (h *DefaultObserverHandler) GetDeviceID() string {
	return h.deviceID
}

func (h *DefaultObserverHandler) SetDeviceID(deviceID string) {
	h.deviceID = deviceID
}

func (h *DefaultObserverHandler) SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error) {
	log.Debugf("SignUp: %v", req)
	h.SetDeviceID(req.DeviceID)
	return coapgwService.CoapSignUpResponse{
		AccessToken:  "access-token",
		UserID:       "1",
		RefreshToken: oauthTest.ValidRefreshToken,
		ExpiresIn:    int64(h.accessTokenLifetime.Seconds()),
		RedirectURI:  "",
	}, nil
}

func (h *DefaultObserverHandler) SignOff() error {
	log.Debugf("SignOff deviceID:%v", h.deviceID)
	return nil
}

func (h *DefaultObserverHandler) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	log.Debugf("SignIn: %v", req)
	return coapgwService.CoapSignInResp{
		ExpiresIn: int64(h.accessTokenLifetime.Seconds()),
	}, nil
}

func (h *DefaultObserverHandler) SignOut(req coapgwService.CoapSignInReq) error {
	log.Debugf("SignOut: %v", req)
	return nil
}

func (h *DefaultObserverHandler) PublishResources(req coapgwTestService.PublishRequest) error {
	log.Debugf("PublishResources: %v", req)
	return nil
}

func (h *DefaultObserverHandler) UnpublishResources(req coapgwTestService.UnpublishRequest) error {
	log.Debugf("UnpublishResources: %v", req)
	return nil
}

func (h *DefaultObserverHandler) RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error) {
	log.Debugf("RefreshToken: %v", req)
	return coapgwService.CoapRefreshTokenResp{
		RefreshToken: oauthTest.ValidRefreshToken,
		AccessToken:  "access-token",
		ExpiresIn:    int64(h.accessTokenLifetime.Seconds()),
	}, nil
}

func (h *DefaultObserverHandler) OnObserveResource(ctx context.Context, deviceID, resourceHref string, notification *pool.Message) error {
	log.Debugf("OnObserveResource: %v%v", deviceID, resourceHref)
	msg := message.ToJson(notification, true, true)
	log.Get().With("notification", msg).Debug("RECEIVED-OBSERVE")
	return nil
}

func (h *DefaultObserverHandler) OnGetResourceContent(ctx context.Context, deviceID, resourceHref string, notification *pool.Message) error {
	log.Debugf("OnGetResourceContent: %v%v", deviceID, resourceHref)
	msg := message.ToJson(notification, true, false)
	log.Get().With("notification", msg).Debug("RECEIVED-GET")
	return nil
}
