package service

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

type Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)

func NewAuthInterceptor() Interceptor {
	return func(ctx context.Context, code codes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SignUp, uri.SignIn:
			return ctx, nil
		}
		e := ctx.Value(&authCtxKey)
		if e == nil {
			return ctx, fmt.Errorf("invalid authorization context")
		}
		authCtx := e.(*authorizationContext)
		err := authCtx.IsValid()
		if err != nil {
			return ctx, err
		}
		return grpc.CtxWithIncomingToken(grpc.CtxWithToken(ctx, authCtx.GetAccessToken()), authCtx.GetAccessToken()), nil
	}
}

func (s *Service) ValidateToken(ctx context.Context, token string) (pkgJwt.Claims, error) {
	ctx, cancel := context.WithTimeout(ctx, s.config.APIs.COAP.KeepAlive.Timeout)
	defer cancel()
	m, err := s.jwtValidator.ParseWithContext(ctx, token)
	if err != nil {
		return nil, err
	}
	return pkgJwt.Claims(m), nil
}

func (s *Service) VerifyDeviceID(tlsDeviceID string, claim pkgJwt.Claims) error {
	if !s.config.APIs.COAP.TLS.Enabled {
		return nil
	}
	if !s.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return nil
	}
	if s.config.Clients.AuthServer.DeviceIDClaim == "" {
		return nil
	}
	jwtDeviceID := claim.DeviceID(s.config.Clients.AuthServer.DeviceIDClaim)
	if jwtDeviceID != tlsDeviceID {
		return fmt.Errorf("deviceID('%v') in JWT doesn't match deviceID from token (%v)", jwtDeviceID, tlsDeviceID)
	}
	return nil
}

func (s *Service) GetDeviceID(claim pkgJwt.Claims, tlsDeviceID, paramDeviceID string) string {
	if s.config.Clients.AuthServer.DeviceIDClaim != "" {
		return claim.DeviceID(s.config.Clients.AuthServer.DeviceIDClaim)
	}
	if s.config.APIs.COAP.TLS.Enabled && s.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return tlsDeviceID
	}
	return paramDeviceID
}
