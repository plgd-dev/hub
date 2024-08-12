package oauthsigner

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/jwt/validator"
	"github.com/plgd-dev/kit/v2/codec/json"
	"go.opentelemetry.io/otel/trace"
)

func setKeyError(key string, err error) error {
	return fmt.Errorf("failed to set %v: %w", key, err)
}

type OAuthSigner struct {
	privateKeyJWTValidators map[string]*validator.Validator
	closer                  fn.FuncList
	config                  Config
	accessTokenKey          interface{}
	accessTokenJwkKey       jwk.Key
}

func New(ctx context.Context, config Config, getOpenIDConfiguration validator.GetOpenIDConfigurationFunc, customTokenIssuerClients map[string]pkgJwt.TokenIssuerClient, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*OAuthSigner, error) {
	accessTokenKey, err := LoadPrivateKey(config.PrivateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("cannot load private privateKeyFile(%v): %w", config.PrivateKeyFile, err)
	}
	accessTokenJwkKey, err := pkgJwt.CreateJwkKey(accessTokenKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create jwk for idToken: %w", err)
	}

	privateKeyJWTValidators := make(map[string]*validator.Validator, len(config.Clients))
	var closer fn.FuncList
	for _, c := range config.Clients {
		if !c.JWTPrivateKey.Enabled {
			continue
		}
		validator, err := validator.New(ctx, c.JWTPrivateKey.Authorization, fileWatcher, logger, tracerProvider, validator.WithGetOpenIDConfiguration(getOpenIDConfiguration), validator.WithCustomTokenIssuerClients(customTokenIssuerClients))
		if err != nil {
			closer.Execute()
			return nil, fmt.Errorf("cannot create validator: %w", err)
		}
		privateKeyJWTValidators[c.ID] = validator
		closer.AddFunc(validator.Close)
	}
	return &OAuthSigner{
		privateKeyJWTValidators: privateKeyJWTValidators,
		closer:                  closer,
		config:                  config,
		accessTokenKey:          accessTokenKey,
		accessTokenJwkKey:       accessTokenJwkKey,
	}, nil
}

func (s *OAuthSigner) GetValidator(clientID string) (*validator.Validator, bool) {
	v, ok := s.privateKeyJWTValidators[clientID]
	return v, ok
}

func (s *OAuthSigner) SignRaw(data []byte) ([]byte, error) {
	hdr := jws.NewHeaders()
	if err := hdr.Set(jws.TypeKey, `JWT`); err != nil {
		return nil, setKeyError(jws.TypeKey, err)
	}
	if err := hdr.Set(jws.KeyIDKey, s.accessTokenJwkKey.KeyID()); err != nil {
		return nil, setKeyError(jws.KeyIDKey, err)
	}

	payload, err := jws.Sign(data, jws.WithKey(s.accessTokenJwkKey.Algorithm(), s.accessTokenKey, jws.WithProtectedHeaders(hdr)))
	if err != nil {
		return nil, fmt.Errorf("failed to create UserToken: %w", err)
	}
	return payload, nil
}

func (s *OAuthSigner) GetJWK() jwk.Key {
	return s.accessTokenJwkKey
}

func (s *OAuthSigner) Sign(token jwt.Token) ([]byte, error) {
	buf, err := json.Encode(token)
	if err != nil {
		return nil, fmt.Errorf("failed to encode token: %w", err)
	}
	return s.SignRaw(buf)
}

func (s *OAuthSigner) Close() {
	s.closer.Execute()
}

func (s *OAuthSigner) GetDomain() string {
	return s.config.GetDomain()
}

func (s *OAuthSigner) GetAuthority() string {
	return s.config.GetAuthority()
}

func (s *OAuthSigner) GetOwnerClaim() string {
	return s.config.OwnerClaim
}

func (s *OAuthSigner) GetDeviceIDClaim() string {
	return s.config.DeviceIDClaim
}

func (s *OAuthSigner) GetClients() OAuthClientsConfig {
	return s.config.Clients
}
