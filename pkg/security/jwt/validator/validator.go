package validator

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/plgd-dev/cloud/pkg/net/http/client"
	jwtValidator "github.com/plgd-dev/cloud/pkg/security/jwt"
	"go.uber.org/zap"
)

// Validator Client.
type Validator struct {
	http      *client.Client
	validator *jwtValidator.Validator
}

// AddCloseFunc adds a function to be called by the Close method.
// This eliminates the need for wrapping the Client.
func (v *Validator) AddCloseFunc(f func()) {
	v.http.AddCloseFunc(f)
}

func (v *Validator) Close() {
	v.http.Close()
}

func New(config Config, logger *zap.Logger) (*Validator, error) {
	http, err := client.New(config.HTTP, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create cert manager: %w", err)
	}
	return &Validator{
		http:      http,
		validator: jwtValidator.NewValidatorWithKeyCache(jwtValidator.NewKeyCacheWithHttp(config.URL, http.HTTP())),
	}, nil
}

func (v *Validator) Parse(token string) (jwt.MapClaims, error) {
	return v.validator.Parse(token)
}

func (v *Validator) ParseWithClaims(token string, claims jwt.Claims) error {
	return v.validator.ParseWithClaims(token, claims)
}
