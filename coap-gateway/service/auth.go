package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/hub/v2/coap-gateway/uri"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
)

type (
	Interceptor = func(ctx context.Context, code codes.Code, path string) (context.Context, error)
	VerifyByCRL = func(context.Context, *x509.Certificate, []string) error
)

func newAuthInterceptor() Interceptor {
	return func(ctx context.Context, _ codes.Code, path string) (context.Context, error) {
		switch path {
		case uri.RefreshToken, uri.SignUp, uri.SignIn, plgdtime.ResourceURI:
			return ctx, nil
		}
		e := ctx.Value(&authCtxKey)
		if e == nil {
			return ctx, errors.New("invalid authorization context")
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

func (s *Service) verifyDeviceID(tlsDeviceID string, claim pkgJwt.Claims) (string, error) {
	jwtDeviceID, err := claim.GetDeviceID(s.config.APIs.COAP.Authorization.DeviceIDClaim)
	if err != nil {
		return "", fmt.Errorf("cannot get device id claim from access token: %w", err)
	}
	if s.config.APIs.COAP.Authorization.DeviceIDClaim != "" && jwtDeviceID == "" {
		return "", fmt.Errorf("access token doesn't contain the required device id claim('%v')", s.config.APIs.COAP.Authorization.DeviceIDClaim)
	}
	if !s.config.APIs.COAP.TLS.IsEnabled() || !s.config.APIs.COAP.TLS.Embedded.ClientCertificateRequired {
		return jwtDeviceID, nil
	}
	if tlsDeviceID == "" {
		return "", errors.New("certificate of device doesn't contain device id")
	}
	if s.config.APIs.COAP.Authorization.DeviceIDClaim != "" && jwtDeviceID != tlsDeviceID {
		return "", fmt.Errorf("access token issued to the device ('%v') used by the different device ('%v')", jwtDeviceID, tlsDeviceID)
	}
	return tlsDeviceID, nil
}

func (s *Service) VerifyAndResolveDeviceID(tlsDeviceID, paramDeviceID string, claim pkgJwt.Claims) (string, error) {
	deviceID, err := s.verifyDeviceID(tlsDeviceID, claim)
	if err != nil {
		return "", err
	}
	if deviceID == "" {
		return paramDeviceID, nil
	}
	return deviceID, nil
}

func verifyChain(ctx context.Context, chain []*x509.Certificate, capool *x509.CertPool, identityPropertiesRequired bool, verifyByCRL VerifyByCRL) error {
	if len(chain) == 0 {
		return errors.New("certificate chain is empty")
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
		return errors.New("the extended key usage field in the device certificate does not contain client authentication")
	}
	if !ekuHasServer {
		return errors.New("the extended key usage field in the device certificate does not contain server authentication")
	}

	if identityPropertiesRequired {
		_, err = coap.GetDeviceIDFromIdentityCertificate(certificate)
		if err != nil {
			return fmt.Errorf("the device ID is not part of the certificate's common name: %w", err)
		}
	}

	if len(certificate.CRLDistributionPoints) > 0 {
		if verifyByCRL == nil {
			return errors.New("cannot verify certificate validity by CRL: verification function not provided")
		}
		if err = verifyByCRL(ctx, certificate, certificate.CRLDistributionPoints); err != nil {
			return err
		}
	}
	return nil
}

func MakeGetConfigForClient(ctx context.Context, tlsCfg *tls.Config, identityPropertiesRequired bool, verifyByCRL VerifyByCRL) tls.Config {
	return tls.Config{
		GetCertificate: tlsCfg.GetCertificate,
		MinVersion:     tlsCfg.MinVersion,
		ClientAuth:     tlsCfg.ClientAuth,
		ClientCAs:      tlsCfg.ClientCAs,
		VerifyPeerCertificate: func(_ [][]byte, chains [][]*x509.Certificate) error {
			var errs *multierror.Error
			for _, chain := range chains {
				err := verifyChain(ctx, chain, tlsCfg.ClientCAs, identityPropertiesRequired, verifyByCRL)
				if err == nil {
					return nil
				}
				errs = multierror.Append(errs, err)
			}
			err := errors.New("empty chains")
			if errs.ErrorOrNil() != nil {
				err = errs
			}
			return pkgX509.NewError(chains, err)
		},
	}
}
