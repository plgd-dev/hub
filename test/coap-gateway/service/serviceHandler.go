package service

import (
	"github.com/plgd-dev/go-coap/v2/tcp"
	coapgwService "github.com/plgd-dev/hub/v2/coap-gateway/service"
)

type ServiceHandlerConfig struct {
	coapConn *tcp.ClientConn
}

func (s *ServiceHandlerConfig) GetCoapConnection() *tcp.ClientConn {
	return s.coapConn
}

type Option interface {
	Apply(o *ServiceHandlerConfig)
}

type CoapConnectionOpt struct {
	coapConn *tcp.ClientConn
}

func (o CoapConnectionOpt) Apply(opts *ServiceHandlerConfig) {
	opts.coapConn = o.coapConn
}

func WithCoapConnectionOpt(c *tcp.ClientConn) CoapConnectionOpt {
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
