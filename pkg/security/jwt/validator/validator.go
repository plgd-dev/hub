package validator

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	jwtValidator "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"go.opentelemetry.io/otel/trace"
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

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Validator, error) {
	httpClient, err := client.New(config.HTTP, fileWatcher, logger, tracerProvider)
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
		validator:           jwtValidator.NewValidator(jwtValidator.NewKeyCache(openIDCfg.JWKSURL, httpClient.HTTP())),
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
