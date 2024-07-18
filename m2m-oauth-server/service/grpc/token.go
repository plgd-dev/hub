package grpc

import (
	"context"
	"errors"
	"fmt"
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

	simpleSets := []struct {
		key string
		val interface{}
	}{
		{jwt.JwtIDKey, tokenReq.id},
		{jwt.SubjectKey, getSubject(clientCfg, tokenReq)},
		{jwt.AudienceKey, tokenReq.host},
		{jwt.IssuedAtKey, tokenReq.issuedAt},
		{uri.ScopeKey, tokenReq.scopes},
		{uri.ClientIDKey, clientCfg.ID},
		{jwt.IssuerKey, tokenReq.host},
	}
	for _, set := range simpleSets {
		if err := token.Set(set.key, set.val); err != nil {
			return nil, setKeyError(set.key, err)
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
	if tokenReq.CreateTokenRequest.GetTokenName() != "" && tokenReq.ownerClaim != "name" {
		return token.Set("name", tokenReq.CreateTokenRequest.GetTokenName())
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
	var expires time.Time
	ttl := time.Duration(tokenReq.CreateTokenRequest.GetTimeToLive()) * time.Nanosecond
	if ttl == 0 || (ttl > clientCfg.AccessTokenLifetime && clientCfg.AccessTokenLifetime > 0) {
		ttl = clientCfg.AccessTokenLifetime
	}
	if ttl > 0 {
		expires = tokenReq.issuedAt.Add(clientCfg.AccessTokenLifetime)
	}
	return expires
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
	host                string                      `json:"-"`
	scopes              string                      `json:"-"`
	ownerClaim          string                      `json:"-"`
	deviceIDClaim       string                      `json:"-"`
	tokenType           oauthsigner.AccessTokenType `json:"-"`
	originalTokenClaims goJwt.MapClaims             `json:"-"`
	issuedAt            time.Time                   `json:"-"`
	expiration          time.Time                   `json:"-"`
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

func (s *M2MOAuthServiceServer) validateTokenRequest(ctx context.Context, clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if err := validateGrantType(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateClient(clientCfg, tokenReq); err != nil {
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
		return fmt.Errorf("client(%v) not found", tokenReq.CreateTokenRequest.GetClientId())
	}
	if clientCfg.Secret != "" && !clientCfg.JWTPrivateKey.Enabled && clientCfg.Secret != tokenReq.CreateTokenRequest.GetClientSecret() {
		return errors.New("invalid client secret")
	}
	return nil
}

func validateGrantType(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	// clientCfg.AllowedGrantTypes is always non-empty
	if !sliceContains(clientCfg.AllowedGrantTypes, []oauthsigner.GrantType{oauthsigner.GrantType(tokenReq.CreateTokenRequest.GetGrantType())}) {
		return fmt.Errorf("invalid grant type(%v)", tokenReq.CreateTokenRequest.GetGrantType())
	}
	return nil
}

func validateAudience(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if !sliceContains(clientCfg.AllowedAudiences, tokenReq.CreateTokenRequest.GetAudience()) {
		return fmt.Errorf("invalid audience(%v)", tokenReq.CreateTokenRequest.GetAudience())
	}
	return nil
}

func validateScopes(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if len(tokenReq.CreateTokenRequest.GetScope()) == 0 {
		tokenReq.CreateTokenRequest.Scope = clientCfg.AllowedScopes
	}
	if !sliceContains(clientCfg.AllowedScopes, tokenReq.CreateTokenRequest.GetScope()) {
		return fmt.Errorf("invalid scope(%v)", tokenReq.CreateTokenRequest.GetScope())
	}
	return nil
}

func validateClientAssertionType(clientCfg *oauthsigner.Client, tokenReq *tokenRequest) error {
	if tokenReq.CreateTokenRequest.GetClientAssertionType() != "" && clientCfg.JWTPrivateKey.Enabled && tokenReq.CreateTokenRequest.GetClientAssertionType() != uri.ClientAssertionTypeJWT {
		return fmt.Errorf("invalid client assertion type(%v)", tokenReq.CreateTokenRequest.GetClientAssertionType())
	}
	return nil
}

func (s *M2MOAuthServiceServer) validateClientAssertion(ctx context.Context, tokenReq *tokenRequest) error {
	if tokenReq.CreateTokenRequest.GetClientAssertion() == "" {
		return nil
	}
	v, ok := s.signer.GetValidator(tokenReq.CreateTokenRequest.GetClientId())
	if !ok {
		return errors.New("invalid client assertion")
	}
	token, err := v.GetParser().ParseWithContext(ctx, tokenReq.CreateTokenRequest.GetClientAssertion())
	if err != nil {
		return fmt.Errorf("invalid client assertion: %w", err)
	}
	tokenReq.originalTokenClaims = token
	claims := pkgJwt.Claims(token)
	owner, err := claims.GetOwner(s.signer.Config.OwnerClaim)
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim owner: %w", err)
	}
	tokenReq.owner = owner
	sub, err := claims.GetSubject()
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim sub: %w", err)
	}
	tokenReq.subject = sub
	if s.signer.Config.DeviceIDClaim == "" {
		return nil
	}
	deviceID, err := claims.GetDeviceID(s.signer.Config.DeviceIDClaim)
	if err == nil {
		tokenReq.deviceID = deviceID
	}
	return nil
}
