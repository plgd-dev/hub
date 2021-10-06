package validator

import (
	"context"
	"fmt"

	"github.com/plgd-dev/cloud/v2/pkg/log"

	"github.com/golang-jwt/jwt/v4"
	"github.com/plgd-dev/cloud/v2/pkg/net/http/client"
	jwtValidator "github.com/plgd-dev/cloud/v2/pkg/security/jwt"
	"github.com/plgd-dev/cloud/v2/pkg/security/openid"
)

// Validator Client.
type Validator struct {
	http                *client.Client
	validator           *jwtValidator.Validator
	openIDConfiguration openid.Config

	// TODO check audience at token
	audience string
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (v *Validator) AddCloseFunc(f func()) {
	v.http.AddCloseFunc(f)
}

func (v *Validator) Close() {
	v.http.Close()
}

func New(ctx context.Context, config Config, logger log.Logger) (*Validator, error) {
	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, config.HTTP.Timeout)
	defer cancel()

	openIDCfg, err := openid.GetConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		httpClient.Close()
		return nil, fmt.Errorf("cannot get openId configuration: %w", err)
	}

	return &Validator{
		http:                httpClient,
		openIDConfiguration: openIDCfg,
		audience:            config.Audience,
		validator:           jwtValidator.NewValidatorWithKeyCache(jwtValidator.NewKeyCacheWithHttp(openIDCfg.JWKSURL, httpClient.HTTP())),
	}, nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.validator.Parse(token)
}

func (v *Validator) OpenIDConfiguration() openid.Config {
	return v.openIDConfiguration
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	return v.validator.ParseWithClaims(token, claims)
}
