package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
)

type Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)

func newAuthInterceptor() Interceptor {
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
	if !s.config.APIs.COAP.TLS.IsEnabled() {
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

func verifyChain(chain []*x509.Certificate, capool *x509.CertPool) error {
	if len(chain) == 0 {
		return fmt.Errorf("certificate chain is empty")
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
		return err
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
		return fmt.Errorf("the extended key usage field in the device certificate does not contain client authentication")
	}
	if !ekuHasServer {
		return fmt.Errorf("the extended key usage field in the device certificate does not contain server authentication")
	}
	_, err = coap.GetDeviceIDFromIdentityCertificate(certificate)
	if err != nil {
		return fmt.Errorf("the device ID is not part of the certificate's common name: %w", err)
	}
	return nil
}

func MakeGetConfigForClient(tlsCfg *tls.Config) tls.Config {
	return tls.Config{
		GetCertificate: tlsCfg.GetCertificate,
		MinVersion:     tlsCfg.MinVersion,
		ClientAuth:     tlsCfg.ClientAuth,
		ClientCAs:      tlsCfg.ClientCAs,
		VerifyPeerCertificate: func(rawCerts [][]byte, chains [][]*x509.Certificate) error {
			var errors *multierror.Error
			for _, chain := range chains {
				err := verifyChain(chain, tlsCfg.ClientCAs)
				if err == nil {
					return nil
				}
				errors = multierror.Append(errors, err)
			}
			err := fmt.Errorf("empty chains")
			if errors.ErrorOrNil() != nil {
				err = errors
			}
			return pkgX509.NewError(chains, err)
		},
	}
}
