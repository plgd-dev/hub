package service

import (
	coapgwService "github.com/plgd-dev/hub/coap-gateway/service"
)

type ServiceHandler interface {
	SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error)
	SignOff() error
	SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error)
	SignOut(req coapgwService.CoapSignInReq) error
	PublishResources(req PublishRequest) error
	UnpublishResources(req UnpublishRequest) error
	RefreshToken(req CoapRefreshTokenReq) (CoapRefreshTokenResp, error)
}
