package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/plgd-dev/cloud/coap-gateway/uri"
	"github.com/plgd-dev/cloud/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/sdk/pkg/net/coap"
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
	jwtDeviceID := claim.DeviceID(s.config.APIs.COAP.Authorization.DeviceIDClaim)
	if s.config.APIs.COAP.Authorization.DeviceIDClaim != "" && jwtDeviceID == "" {
		return fmt.Errorf("access token doesn't contain the required device id claim('%v')", s.config.APIs.COAP.Authorization.DeviceIDClaim)
	}
	if !s.config.APIs.COAP.TLS.Enabled {
		return nil
	}
	if !s.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return nil
	}
	if tlsDeviceID == "" {
		return fmt.Errorf("certificate of device doesn't contains device id")
	}
	if s.config.APIs.COAP.Authorization.DeviceIDClaim == "" {
		return nil
	}
	if jwtDeviceID != tlsDeviceID {
		return fmt.Errorf("access token issued to the device ('%v') used by the different device ('%v')", jwtDeviceID, tlsDeviceID)
	}
	return nil
}

func verifyChain(chain []*x509.Certificate, capool *x509.CertPool) (string, error) {
	if len(chain) == 0 {
		return "", fmt.Errorf("empty chain")
	}
	certificate := chain[0]
	intermediateCAPool := x509.NewCertPool()
	for i := 1; i < len(chain); i++ {
		intermediateCAPool.AddCert(chain[i])
	}
	_, err := certificate.Verify(x509.VerifyOptions{
		Roots:         capool,
		Intermediates: intermediateCAPool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		return "", err
	}
	// verify EKU manually
	ekuHasClient := false
	ekuHasServer := false
	for _, eku := range certificate.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			ekuHasClient = true
		}
		if eku == x509.ExtKeyUsageServerAuth {
			ekuHasServer = true
		}
	}
	if !ekuHasClient {
		return "", fmt.Errorf("not contains ExtKeyUsageClientAuth")
	}
	if !ekuHasServer {
		return "", fmt.Errorf("not contains ExtKeyUsageServerAuth")
	}
	return coap.GetDeviceIDFromIndetityCertificate(certificate)
}

func MakeGetConfigForClient(tlsCfg *tls.Config, deviceIdCache *cache.Cache) func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
	return func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
		return &tls.Config{
			GetCertificate: tlsCfg.GetCertificate,
			MinVersion:     tlsCfg.MinVersion,
			ClientAuth:     tlsCfg.ClientAuth,
			ClientCAs:      tlsCfg.ClientCAs,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				var errors []error
				for _, chain := range verifiedChains {
					deviceID, err := verifyChain(chain, tlsCfg.ClientCAs)
					if err == nil {
						deviceIdCache.SetDefault(chi.Conn.RemoteAddr().String(), deviceID)
						return nil
					}
					errors = append(errors, err)
				}
				if len(errors) > 0 {
					return fmt.Errorf("%v", errors)
				}
				return fmt.Errorf("empty chains")
			},
		}, nil
	}
}
