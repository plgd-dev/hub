package validator

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
	jwtValidator "github.com/plgd-dev/cloud/pkg/security/jwt"
	"github.com/plgd-dev/kit/codec/json"
	"go.uber.org/zap"
)

// Validator Client.
type Validator struct {
	http                *client.Client
	validator           *jwtValidator.Validator
	openIDConfiguration OpenIDConfiguration

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

type OpenIDConfiguration struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	JWKSURL     string   `json:"jwks_uri"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Algorithms  []string `json:"id_token_signing_alg_values_supported"`
}

func (c OpenIDConfiguration) Validate() error {
	if c.JWKSURL == "" {
		return fmt.Errorf("jwks_uri('%v')", c.JWKSURL)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("token_endpoint('%v')", c.TokenURL)
	}
	if c.AuthURL == "" {
		return fmt.Errorf("authorization_endpoint('%v')", c.AuthURL)
	}
	if c.Issuer == "" {
		return fmt.Errorf("issuer('%v')", c.Issuer)
	}
	return nil
}

func GetOpenIDConfiguration(ctx context.Context, httpClient *http.Client, domain string) (OpenIDConfiguration, error) {
	href := domain + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, href, nil)
	if err != nil {
		return OpenIDConfiguration{}, fmt.Errorf("cannot create request for GET %v: %w", href, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return OpenIDConfiguration{}, fmt.Errorf("cannot GET %v: %w", href, err)
	}
	if resp.Body == nil {
		return OpenIDConfiguration{}, fmt.Errorf("invalid response GET %v response: is empty", href)
	}
	defer resp.Body.Close()
	var cfg OpenIDConfiguration
	err = json.ReadFrom(resp.Body, &cfg)
	if err != nil {
		return OpenIDConfiguration{}, fmt.Errorf("cannot decode GET %v response: %w", href, err)
	}
	err = cfg.Validate()
	if err != nil {
		return OpenIDConfiguration{}, fmt.Errorf("invalid property of GET %v response: %w", href, err)
	}

	return cfg, nil
}

func New(ctx context.Context, config Config, logger *zap.Logger) (*Validator, error) {
	httpClient, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, config.HTTP.Timeout)
	defer cancel()

	openIDCfg, err := GetOpenIDConfiguration(ctx, httpClient.HTTP(), config.Authority)
	if err != nil {
		httpClient.Close()
		return nil, fmt.Errorf("cannot get openid configuration: %w", err)
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

func (v *Validator) OpenIDConfiguration() OpenIDConfiguration {
	return v.openIDConfiguration
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	return v.validator.ParseWithClaims(token, claims)
}
