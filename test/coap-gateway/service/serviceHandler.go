package service

import (
	"github.com/plgd-dev/go-coap/v3/tcp/client"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
)

type ServiceHandlerConfig struct {
	coapConn *client.ClientConn
}

func (s *ServiceHandlerConfig) GetCoapConnection() *client.ClientConn {
	return s.coapConn
}

type Option interface {
	Apply(o *ServiceHandlerConfig)
}

type CoapConnectionOpt struct {
	coapConn *client.ClientConn
}

func (o CoapConnectionOpt) Apply(opts *ServiceHandlerConfig) {
	opts.coapConn = o.coapConn
}

func WithCoapConnectionOpt(c *client.ClientConn) CoapConnectionOpt {
	return CoapConnectionOpt{
		coapConn: c,
	}
}

type MakeServiceHandler = func(service *Service, opts ...Option) ServiceHandler

type VerifyServiceHandler = func(ServiceHandler)

type ServiceHandler interface {
	SignUp(req coapgwService.CoapSignUpRequest) (coapgwService.CoapSignUpResponse, error)
	SignOff() error
	SignIn(req coapgwService.CoapSignInReq) (coapgwService.CoapSignInResp, error)
	SignOut(req coapgwService.CoapSignInReq) error
	PublishResources(req PublishRequest) error
	UnpublishResources(req UnpublishRequest) error
	RefreshToken(req coapgwService.CoapRefreshTokenReq) (coapgwService.CoapRefreshTokenResp, error)
}
