package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goJwt "github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	oauthsigner "github.com/plgd-dev/hub/v2/m2m-oauth-server/oauthSigner"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
)

func setKeyError(key string, err error) error {
	return fmt.Errorf("failed to set %v: %w", key, err)
}

func setKeyErrorExt(key, info interface{}, err error) error {
	return fmt.Errorf("failed to set %v('%v'): %w", key, info, err)
}

func makeAccessToken(clientCfg *oauthsigner.Client, tokenReq tokenRequest) (jwt.Token, error) {
	token := jwt.New()

	claims := map[string]interface{}{
		jwt.JwtIDKey:    tokenReq.id,
		jwt.SubjectKey:  tokenReq.subject,
		jwt.AudienceKey: strings.Join(tokenReq.Audience, " "),
		jwt.IssuedAtKey: tokenReq.issuedAt,
		uri.ScopeKey:    tokenReq.scopes,
		uri.ClientIDKey: clientCfg.ID,
		jwt.IssuerKey:   tokenReq.issuer,
	}
	for key, val := range claims {
		if err := token.Set(key, val); err != nil {
			return nil, setKeyError(key, err)
		}
	}
	if !tokenReq.expiration.IsZero() {
		if err := token.Set(jwt.ExpirationKey, tokenReq.expiration); err != nil {
			return nil, setKeyError(jwt.ExpirationKey, err)
		}
	}
	if err := setDeviceIDClaim(token, tokenReq); err != nil {
		return nil, err
	}
	if err := setName(token, tokenReq); err != nil {
		return nil, err
	}
	if err := setOwnerClaim(token, tokenReq); err != nil {
		return nil, err
	}
	if err := setOriginTokenClaims(token, tokenReq); err != nil {
		return nil, err
	}

	for k, v := range clientCfg.InsertTokenClaims {
		if _, ok := token.Get(k); ok {
			continue
		}
		if err := token.Set(k, v); err != nil {
			return nil, setKeyErrorExt(k, v, err)
		}
	}

	return token, nil
}

func getSubject(clientCfg *oauthsigner.Client, tokenReq tokenRequest) string {
	if tokenReq.subject != "" {
		return tokenReq.subject
	}
	if tokenReq.owner != "" {
		return tokenReq.owner
	}
	return clientCfg.ID
}

func setDeviceIDClaim(token jwt.Token, tokenReq tokenRequest) error {
	if tokenReq.deviceID != "" && tokenReq.deviceIDClaim != "" {
		return token.Set(tokenReq.deviceIDClaim, tokenReq.deviceID)
	}
	return nil
}

func setOwnerClaim(token jwt.Token, tokenReq tokenRequest) error {
	if tokenReq.owner != "" && tokenReq.ownerClaim != "" {
		return token.Set(tokenReq.ownerClaim, tokenReq.owner)
	}
	return nil
}

func setName(token jwt.Token, tokenReq tokenRequest) error {
	if tokenReq.GetTokenName() != "" && tokenReq.ownerClaim != "name" {
		return token.Set("name", tokenReq.GetTokenName())
	}
	return nil
}

func setOriginTokenClaims(token jwt.Token, tokenReq tokenRequest) error {
	if len(tokenReq.originalTokenClaims) > 0 {
		return token.Set(uri.OriginalTokenClaims, tokenReq.originalTokenClaims)
	}
	return nil
}

func getExpirationTime(clientCfg *oauthsigner.Client, tokenReq tokenRequest) time.Time {
	var wantExpiration time.Time
	if tokenReq.GetExpiration() > 0 {
		wantExpiration = time.Unix(tokenReq.GetExpiration(), 0)
	}
	if !wantExpiration.IsZero() && tokenReq.issuedAt.After(wantExpiration) {
		return time.Time{}
	}
	if clientCfg.AccessTokenLifetime == 0 {
		if !wantExpiration.IsZero() {
			return wantExpiration
		}
		return time.Time{}
	}
	if wantExpiration.IsZero() {
		return tokenReq.issuedAt.Add(clientCfg.AccessTokenLifetime)
	}
	clientExpiration := tokenReq.issuedAt.Add(clientCfg.AccessTokenLifetime)
	if clientExpiration.Before(wantExpiration) {
		return clientExpiration
	}
	return wantExpiration
}

func (s *M2MOAuthServiceServer) generateAccessToken(clientCfg *oauthsigner.Client, tokenReq tokenRequest) (string, error) {
	token, err := makeAccessToken(clientCfg, tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to make token: %w", err)
	}
	payload, err := s.signer.Sign(token)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return string(payload), nil
}

type tokenRequest struct {
	*pb.CreateTokenRequest

	id                  string                      `json:"-"`
	deviceID            string                      `json:"-"`
	owner               string                      `json:"-"`
	subject             string                      `json:"-"`
	scopes              string                      `json:"-"`
	ownerClaim          string                      `json:"-"`
	deviceIDClaim       string                      `json:"-"`
	tokenType           oauthsigner.AccessTokenType `json:"-"`
	originalTokenClaims goJwt.MapClaims             `json:"-"`
	issuedAt            time.Time                   `json:"-"`
	expiration          time.Time                   `json:"-"`
	issuer              string                      `json:"-"`
}

func sliceContains[T comparable](s []T, sub []T) bool {
	// sub must be non-empty
	if len(s) > 0 && len(sub) == 0 {
		return false
	}
	check := make(map[T]struct{}, len(sub))
	for _, e := range sub {
		check[e] = struct{}{}
	}
	for _, e := range s {
		delete(check, e)
	}
	return len(check) == 0
}

func validateExpiration(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if tokenReq.GetExpiration() > 0 {
		if tokenReq.GetExpiration() < tokenReq.issuedAt.Unix() {
			return fmt.Errorf("expiration(%v) must be greater than issuedAt(%v)", time.Unix(tokenReq.GetExpiration(), 0), tokenReq.issuedAt)
		}
		if clientCfg.AccessTokenLifetime > 0 {
			if tokenReq.GetExpiration() > tokenReq.issuedAt.Add(clientCfg.AccessTokenLifetime).Unix() {
				return fmt.Errorf("expiration(%v) must be less than or equal to issuedAt + client accessTokenLifetime(%v)", time.Unix(tokenReq.GetExpiration(), 0), tokenReq.issuedAt.Add(clientCfg.AccessTokenLifetime))
			}
		}
	}
	return nil
}

func (s *M2MOAuthServiceServer) validateTokenRequest(ctx context.Context, clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if err := validateGrantType(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateClient(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateExpiration(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateClientAssertionType(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := s.validateClientAssertion(ctx, tokenReq); err != nil {
		return err
	}
	if err := validateAudience(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateScopes(clientCfg, tokenReq); err != nil {
		return err
	}
	return nil
}

func validateClient(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if clientCfg == nil {
		return fmt.Errorf("client(%v) not found", tokenReq.GetClientId())
	}
	if clientCfg.Secret != "" && !clientCfg.JWTPrivateKey.Enabled && clientCfg.Secret != tokenReq.GetClientSecret() {
		return errors.New("invalid client secret")
	}
	return nil
}

func validateGrantType(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	// clientCfg.AllowedGrantTypes is always non-empty
	if !sliceContains(clientCfg.AllowedGrantTypes, []oauthsigner.GrantType{oauthsigner.GrantType(tokenReq.GetGrantType())}) {
		return fmt.Errorf("invalid grant type(%v)", tokenReq.GetGrantType())
	}
	return nil
}

func validateAudience(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if !sliceContains(clientCfg.AllowedAudiences, tokenReq.GetAudience()) {
		return fmt.Errorf("invalid audience(%v)", tokenReq.GetAudience())
	}
	return nil
}

func validateScopes(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if len(tokenReq.GetScope()) == 0 {
		tokenReq.Scope = clientCfg.AllowedScopes
	}
	if !sliceContains(clientCfg.AllowedScopes, tokenReq.GetScope()) {
		return fmt.Errorf("invalid scope(%v)", tokenReq.GetScope())
	}
	return nil
}

func validateClientAssertionType(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if tokenReq.GetClientAssertionType() != "" && clientCfg.JWTPrivateKey.Enabled && tokenReq.GetClientAssertionType() != uri.ClientAssertionTypeJWT {
		return fmt.Errorf("invalid client assertion type(%v)", tokenReq.GetClientAssertionType())
	}
	return nil
}

func (s *M2MOAuthServiceServer) validateClientAssertion(ctx context.Context, tokenReq *tokenRequest) error {
	if tokenReq.GetClientAssertion() == "" {
		return nil
	}
	v, ok := s.signer.GetValidator(tokenReq.GetClientId())
	if !ok {
		return errors.New("invalid client assertion")
	}
	token, err := v.GetParser().ParseWithContext(ctx, tokenReq.GetClientAssertion())
	if err != nil {
		return fmt.Errorf("invalid client assertion: %w", err)
	}
	tokenReq.originalTokenClaims = token
	claims := pkgJwt.Claims(token)
	owner, err := claims.GetOwner(s.signer.GetOwnerClaim())
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim owner: %w", err)
	}
	tokenReq.owner = owner
	sub, err := claims.GetSubject()
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim sub: %w", err)
	}
	tokenReq.subject = sub
	if s.signer.GetDeviceIDClaim() == "" {
		return nil
	}
	deviceID, err := claims.GetDeviceID(s.signer.GetDeviceIDClaim())
	if err == nil {
		tokenReq.deviceID = deviceID
	}
	return nil
}
