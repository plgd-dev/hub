package test

import (
	"context"
	"time"

	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
	"github.com/plgd-dev/hub/v2/coap-gateway/service/message"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/resource-aggregate/commands"
	coapgwTestService "github.com/plgd-dev/hub/v2/test/coap-gateway/service"
	oauthTest "github.com/plgd-dev/hub/v2/test/oauth-server/test"
)

// Default test observer handler
//
// It implements ServiceHandler interface by just logging the called method and
// returning default response and no error (if required).
type DefaultObserverHandler struct {
	deviceID            string
	accessToken         string
	refreshToken        string
	accessTokenLifetime int64 // lifetime in seconds or <0 value for a token without expiration
}

func MakeDefaultObserverHandler(accessTokenLifetime int64) DefaultObserverHandler {
	return DefaultObserverHandler{
		accessToken:         "access-token",
		refreshToken:        oauthTest.ValidRefreshToken,
		accessTokenLifetime: accessTokenLifetime,
	}
}

func (h *DefaultObserverHandler) GetDeviceID() string {
	return h.deviceID
}

func (h *DefaultObserverHandler) SetDeviceID(deviceID string) {
	h.deviceID = deviceID
}

func (h *DefaultObserverHandler) SetAccessToken(accessToken string) {
	h.accessToken = accessToken
}

func (h *DefaultObserverHandler) SetRefreshToken(refreshToken string) {
	h.refreshToken = refreshToken
}

func (h *DefaultObserverHandler) SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error) {
	log.Debugf("SignUp: %v", req)
	h.SetDeviceID(req.DeviceID)
	return coapgwService.CoapSignUpResponse{
		AccessToken:  h.accessToken,
		UserID:       "1",
		RefreshToken: h.refreshToken,
		ExpiresIn:    h.accessTokenLifetime,
		RedirectURI:  "",
	}, nil
}

func (h *DefaultObserverHandler) CloseOnError() bool {
	return true
}

func (h *DefaultObserverHandler) SignOff() error {
	log.Debugf("SignOff deviceID:%v", h.deviceID)
	return nil
}

func (h *DefaultObserverHandler) SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error) {
	log.Debugf("SignIn: %v", req)
	return coapgwService.CoapSignInResp{
		ExpiresIn: h.accessTokenLifetime,
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
		RefreshToken: h.refreshToken,
		AccessToken:  h.accessToken,
		ExpiresIn:    h.accessTokenLifetime,
	}, nil
}

func (h *DefaultObserverHandler) OnObserveResource(_ context.Context, deviceID, resourceHref string, resourceTypes []string, notification *pool.Message) error {
	log.Debugf("OnObserveResource: %v%v %v", deviceID, resourceHref, resourceTypes)
	msg := message.ToJson(notification, true, true)
	log.Get().With("notification", msg).Debug("RECEIVED-OBSERVE")
	return nil
}

func (h *DefaultObserverHandler) OnGetResourceContent(_ context.Context, deviceID, resourceHref string, resourceTypes []string, notification *pool.Message) error {
	log.Debugf("OnGetResourceContent: %v%v %v", deviceID, resourceHref, resourceTypes)
	msg := message.ToJson(notification, true, false)
	log.Get().With("notification", msg).Debug("RECEIVED-GET")
	return nil
}

func (h *DefaultObserverHandler) UpdateTwinSynchronization(_ context.Context, deviceID string, status commands.TwinSynchronization_State, t time.Time) error {
	log.Debugf("UpdateTwinSynchronizationStatus: %v %v %v", deviceID, status, t)
	return nil
}
