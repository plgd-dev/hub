package pb

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/security/oauth/clientcredentials"
	"github.com/plgd-dev/hub/v2/device-provisioning-service/uri"
	"github.com/plgd-dev/hub/v2/pkg/config/property/urischeme"
	"github.com/plgd-dev/hub/v2/pkg/net/grpc/client"
	pkgHttpClient "github.com/plgd-dev/hub/v2/pkg/net/http/client"
	pkgCertManagerClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/strings"
	"github.com/plgd-dev/kit/v2/security"
)

type Hubs []*Hub

func (p Hubs) Sort() {
	sort.Slice(p, func(i, j int) bool {
		return p[i].GetId() < p[j].GetId()
	})
}

func (c *TlsConfig) ToConfig() pkgCertManagerClient.Config {
	return pkgCertManagerClient.Config{
		CAPool:          c.GetCaPool(),
		KeyFile:         urischeme.URIScheme(c.GetKey()),
		CertFile:        urischeme.URIScheme(c.GetCert()),
		UseSystemCAPool: c.GetUseSystemCaPool(),
	}
}

func (c *GrpcConnectionConfig) ToConfig() client.Config {
	return client.Config{
		Addr: c.GetAddress(),
		KeepAlive: client.KeepAliveConfig{
			Time:                time.Duration(c.GetKeepAlive().GetTime()) * time.Nanosecond,
			Timeout:             time.Duration(c.GetKeepAlive().GetTimeout()) * time.Nanosecond,
			PermitWithoutStream: c.GetKeepAlive().GetPermitWithoutStream(),
		},
		TLS: c.GetTls().ToConfig(),
	}
}

func (c *AuthorizationProviderConfig) ToConfig() clientcredentials.Config {
	return clientcredentials.Config{
		Authority:        c.GetAuthority(),
		ClientID:         c.GetClientId(),
		ClientSecretFile: urischeme.URIScheme(c.GetClientSecret()),
		Scopes:           c.GetScopes(),
		Audience:         c.GetAudience(),
		HTTP: pkgHttpClient.Config{
			MaxIdleConns:        int(c.GetHttp().GetMaxIdleConns()),
			MaxConnsPerHost:     int(c.GetHttp().GetMaxConnsPerHost()),
			MaxIdleConnsPerHost: int(c.GetHttp().GetMaxIdleConnsPerHost()),
			IdleConnTimeout:     time.Duration(c.GetHttp().GetIdleConnTimeout()) * time.Nanosecond,
			Timeout:             time.Duration(c.GetHttp().GetTimeout()) * time.Nanosecond,
			TLS:                 c.GetHttp().GetTls().ToConfig(),
		},
		// TokenURL: ,
	}
}

func (c *TlsConfig) validateCAPool() error {
	if c.GetCaPool() == nil && !c.GetUseSystemCaPool() {
		return fmt.Errorf("caPool('%v'),useSystemCAPool('%v') - are not set", c.GetCaPool(), c.GetUseSystemCaPool())
	}
	if c.GetCaPool() != nil {
		for i, ca := range c.GetCaPool() {
			data, err := urischeme.URIScheme(ca).Read()
			if err != nil {
				return fmt.Errorf("caPool[%d]('%v') - %w", i, ca, err)
			}
			_, err = security.ParseX509FromPEM(data)
			if err != nil {
				return fmt.Errorf("caPool[%d]('%v') - %w", i, ca, err)
			}
		}
	}
	return nil
}

func (c *TlsConfig) Validate() error {
	if err := c.validateCAPool(); err != nil {
		return err
	}
	if c.GetCert() != "" || c.GetKey() != "" {
		certPEMBlock, err := urischeme.URIScheme(c.GetCert()).Read()
		if err != nil {
			return fmt.Errorf("certFile('%v') - %w", c.GetCert(), err)
		}
		keyPEMBlock, err := urischeme.URIScheme(c.GetKey()).Read()
		if err != nil {
			return fmt.Errorf("keyFile('%v') - %w", c.GetKey(), err)
		}
		_, err = tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return fmt.Errorf("certFile('%v'),keyFile('%v') - %w", c.GetCert(), c.GetKey(), err)
		}
	}
	return nil
}

func (c *GrpcConnectionConfig) Validate() error {
	if c.GetAddress() == "" {
		return fmt.Errorf("address('%v')", c.GetAddress())
	}
	if c.GetTls() == nil {
		return errors.New("tls - is empty")
	}
	if err := c.GetTls().Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

func (c *GrpcClientConfig) Validate() error {
	if c.GetGrpc() == nil {
		return errors.New("grpc - is empty")
	}
	if err := c.GetGrpc().Validate(); err != nil {
		return fmt.Errorf("grpc.%w", err)
	}
	return nil
}

func (c *HttpConfig) Validate() error {
	if c.GetTimeout() < 0 {
		return fmt.Errorf("timeout('%v')", c.GetTimeout())
	}
	if c.GetIdleConnTimeout() < 0 {
		return fmt.Errorf("idleConnTimeout('%v')", c.GetIdleConnTimeout())
	}
	if c.GetTls() == nil {
		return errors.New("tls - is empty")
	}
	if err := c.GetTls().Validate(); err != nil {
		return fmt.Errorf("tls.%w", err)
	}
	return nil
}

func (c *AuthorizationProviderConfig) Validate() error {
	if c.GetName() == "" {
		return fmt.Errorf("name('%v')", c.GetName())
	}
	if c.GetAuthority() == "" {
		return fmt.Errorf("authority('%v')", c.GetAuthority())
	}
	if c.GetClientId() == "" {
		return fmt.Errorf("clientId('%v')", c.GetClientId())
	}
	if _, err := urischeme.URIScheme(c.GetClientSecret()).Read(); err != nil {
		return fmt.Errorf("clientSecret('%v') - %w", c.GetClientSecret(), err)
	}
	if c.GetHttp() == nil {
		return errors.New("http - is empty")
	}
	if err := c.GetHttp().Validate(); err != nil {
		return fmt.Errorf("http.%w", err)
	}
	return nil
}

func (c *AuthorizationConfig) Validate() error {
	if c.GetOwnerClaim() == "" {
		return fmt.Errorf("ownerClaim('%v')", c.GetOwnerClaim())
	}
	if c.GetProvider() == nil {
		return errors.New("provider - is empty")
	}
	if err := c.GetProvider().Validate(); err != nil {
		return fmt.Errorf("provider.%w", err)
	}
	return nil
}

func ValidateCoapGatewayURI(coapGwURI string) (string, error) {
	parsedURL, err := url.Parse(coapGwURI)
	if err == nil {
		switch schema.Scheme(parsedURL.Scheme) {
		case schema.TCPSecureScheme, schema.UDPSecureScheme, schema.TCPScheme, schema.UDPScheme:
			return coapGwURI, nil
		}
	}
	u := uri.CoAPsTCPSchemePrefix + coapGwURI
	parsedURL, err = url.Parse(u)
	if err != nil || parsedURL.Scheme == "" {
		return "", fmt.Errorf("invalid URI('%v')", coapGwURI)
	}
	return u, nil
}

func (h *Hub) normalizeGateways() error {
	for i, gw := range h.GetGateways() {
		if gw == "" {
			return fmt.Errorf("coapGateways[%d]('%v') - is empty", i, gw)
		}
		fixedGw, err := ValidateCoapGatewayURI(gw)
		if err != nil {
			return fmt.Errorf("coapGateways[%d]('%v') - %w", i, gw, err)
		}
		h.Gateways[i] = fixedGw
	}
	h.Gateways = strings.UniqueStable(h.GetGateways())
	return nil
}

func (h *Hub) Validate(owner string) error {
	if h.GetId() == "" {
		return fmt.Errorf("id('%v')", h.GetId())
	}
	if h.GetOwner() == "" {
		return fmt.Errorf("owner('%v') - is empty", h.GetOwner())
	}
	if owner != "" && owner != h.GetOwner() {
		return fmt.Errorf("owner('%v') - expects %v", h.GetOwner(), owner)
	}
	if len(h.GetGateways()) == 0 {
		return errors.New("coapGateways - is empty")
	}
	if err := h.normalizeGateways(); err != nil {
		return err
	}
	if h.GetCertificateAuthority() == nil {
		return errors.New("certificateAuthority - is empty")
	}
	if err := h.GetCertificateAuthority().Validate(); err != nil {
		return fmt.Errorf("certificateAuthority.%w", err)
	}
	if h.GetAuthorization() == nil {
		return errors.New("authorization - is empty")
	}
	if err := h.GetAuthorization().Validate(); err != nil {
		return fmt.Errorf("authorization.%w", err)
	}
	if h.GetName() == "" {
		// for backward compatibility
		h.Name = h.GetId()
	}
	if h.GetHubId() == "" {
		// for backward compatibility
		h.HubId = h.GetId()
	}
	return nil
}
