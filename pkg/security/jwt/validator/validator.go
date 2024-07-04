package validator

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/net/http/client"
	jwtValidator "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"go.opentelemetry.io/otel/trace"
)

// Validator Client.
type Validator struct {
	validator            *jwtValidator.Validator
	openIDConfigurations []openid.Config
	onClose              fn.FuncList

	// TODO check audience at token
	audience string
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (v *Validator) AddCloseFunc(f func()) {
	v.onClose.AddFunc(f)
}

func (v *Validator) Close() {
	v.onClose.Execute()
}

func (v *Validator) GetParser() *jwtValidator.Validator {
	return v.validator
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider) (*Validator, error) {
	keys := jwtValidator.NewMultiKeyCache()
	var onClose fn.FuncList
	openIDConfigurations := make([]openid.Config, 0, len(config.Endpoints))
	for _, authority := range config.Endpoints {
		httpClient, err := client.New(authority.HTTP, fileWatcher, logger, tracerProvider)
		if err != nil {
			return nil, fmt.Errorf("cannot create client cert manager: %w", err)
		}

		ctx2, cancel := context.WithTimeout(ctx, authority.HTTP.Timeout)
		defer cancel()

		openIDCfg, err := openid.GetConfiguration(ctx2, httpClient.HTTP(), authority.Address)
		if err != nil {
			onClose.Execute()
			httpClient.Close()
			return nil, fmt.Errorf("cannot get openId configuration: %w", err)
		}
		onClose.AddFunc(httpClient.Close)
		keys.Add(openIDCfg.Issuer, openIDCfg.JWKSURL, httpClient.HTTP())
		openIDConfigurations = append(openIDConfigurations, openIDCfg)
	}

	return &Validator{
		openIDConfigurations: openIDConfigurations,
		validator:            jwtValidator.NewValidator(keys),
		audience:             config.Audience,
	}, nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.validator.Parse(token)
}

func (v *Validator) OpenIDConfiguration() []openid.Config {
	return v.openIDConfigurations
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	return v.validator.ParseWithClaims(token, claims)
}
