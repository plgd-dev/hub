package validator

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/plgd-dev/hub/v2/pkg/fn"
	"github.com/plgd-dev/hub/v2/pkg/fsnotify"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgHttpUri "github.com/plgd-dev/hub/v2/pkg/net/http/uri"
	cmClient "github.com/plgd-dev/hub/v2/pkg/security/certManager/client"
	"github.com/plgd-dev/hub/v2/pkg/security/certManager/general"
	jwtValidator "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	pkgX509 "github.com/plgd-dev/hub/v2/pkg/security/x509"
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

type GetOpenIDConfigurationFunc func(ctx context.Context, c *http.Client, authority string) (openid.Config, error)

type Options struct {
	getOpenIDConfiguration              GetOpenIDConfigurationFunc
	customTokenIssuerClients            map[string]jwtValidator.TokenIssuerClient
	customDistributionPointVerification pkgX509.CustomDistributionPointVerification
}

func WithGetOpenIDConfiguration(f GetOpenIDConfigurationFunc) func(o *Options) {
	return func(o *Options) {
		o.getOpenIDConfiguration = f
	}
}

func WithCustomTokenIssuerClients(clients map[string]jwtValidator.TokenIssuerClient) func(o *Options) {
	return func(o *Options) {
		o.customTokenIssuerClients = clients
	}
}

func WithCustomDistributionPointVerification(dpVerification pkgX509.CustomDistributionPointVerification) func(o *Options) {
	return func(o *Options) {
		o.customDistributionPointVerification = dpVerification
	}
}

func New(ctx context.Context, config Config, fileWatcher *fsnotify.Watcher, logger log.Logger, tracerProvider trace.TracerProvider, opts ...func(o *Options)) (*Validator, error) {
	options := Options{
		getOpenIDConfiguration:   openid.GetConfiguration,
		customTokenIssuerClients: make(map[string]jwtValidator.TokenIssuerClient),
	}
	for _, o := range opts {
		o(&options)
	}
	if options.getOpenIDConfiguration == nil {
		return nil, errors.New("GetOpenIDConfiguration is nil")
	}
	if options.customTokenIssuerClients == nil {
		return nil, errors.New("customTokenIssuerClients is nil")
	}
	if len(config.Endpoints) == 0 {
		return nil, errors.New("no endpoints")
	}

	cmOptions := []general.SetOption{}
	if options.customDistributionPointVerification != nil {
		cmOptions = append(cmOptions, general.WithCustomDistributionPointVerification(options.customDistributionPointVerification))
	}
	keys := jwtValidator.NewMultiKeyCache()
	var onClose fn.FuncList
	openIDConfigurations := make([]openid.Config, 0, len(config.Endpoints))
	clients := make(map[string]jwtValidator.TokenIssuerClient, len(config.Endpoints))
	for _, authority := range config.Endpoints {
		httpClient, err := cmClient.NewHTTPClient(&authority.HTTP, fileWatcher, logger, tracerProvider, cmOptions...)
		if err != nil {
			onClose.Execute()
			return nil, fmt.Errorf("cannot create client cert manager: %w", err)
		}

		ctx2, cancel := context.WithTimeout(ctx, authority.HTTP.Timeout)
		defer cancel()

		openIDCfg, err := options.getOpenIDConfiguration(ctx2, httpClient.HTTP(), authority.Authority)
		if err != nil {
			onClose.Execute()
			httpClient.Close()
			return nil, fmt.Errorf("cannot get openId configuration: %w", err)
		}
		onClose.AddFunc(httpClient.Close)
		issuer := pkgHttpUri.CanonicalURI(openIDCfg.Issuer)
		keys.Add(issuer, openIDCfg.JWKSURL, httpClient.HTTP())
		openIDConfigurations = append(openIDConfigurations, openIDCfg)
		if openIDCfg.PlgdTokensEndpoint != "" {
			if tokenIssuer, ok := options.customTokenIssuerClients[issuer]; ok {
				clients[issuer] = tokenIssuer
				continue
			}
			clients[issuer] = jwtValidator.NewHTTPClient(httpClient.HTTP(), openIDCfg.PlgdTokensEndpoint)
		}
	}

	var vopts []jwtValidator.Option
	if len(clients) > 0 {
		vopts = append(vopts, jwtValidator.WithTrustVerification(clients, config.TokenVerification.CacheExpiration, ctx.Done()))
	}

	return &Validator{
		openIDConfigurations: openIDConfigurations,
		validator:            jwtValidator.NewValidator(keys, logger, vopts...),
		audience:             config.Audience,
	}, nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.validator.Parse(token)
}

func (v *Validator) OpenIDConfiguration() []openid.Config {
	return v.openIDConfigurations
}

func (v *Validator) ParseWithClaims(ctx context.Context, token string, claims jwt.Claims) error {
	return v.validator.ParseWithClaims(ctx, token, claims)
}
