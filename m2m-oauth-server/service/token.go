package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	goJwt "github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	pkgJwt "github.com/plgd-dev/hub/v2/pkg/security/jwt"
	"github.com/plgd-dev/kit/v2/codec/json"
)

func setKeyError(key string, err error) error {
	return fmt.Errorf("failed to set %v: %w", key, err)
}

func setKeyErrorExt(key, info interface{}, err error) error {
	return fmt.Errorf("failed to set %v('%v'): %w", key, info, err)
}

func makeAccessToken(clientCfg *Client, tokenReq tokenRequest, issuedAt, expires time.Time) (jwt.Token, error) {
	token := jwt.New()

	sub := getSubject(clientCfg, tokenReq)
	if err := token.Set(jwt.SubjectKey, sub); err != nil {
		return nil, setKeyError(jwt.SubjectKey, err)
	}
	if err := token.Set(jwt.AudienceKey, tokenReq.host); err != nil {
		return nil, setKeyError(jwt.AudienceKey, err)
	}
	if err := token.Set(jwt.IssuedAtKey, issuedAt); err != nil {
		return nil, setKeyError(jwt.IssuedAtKey, err)
	}
	if !expires.IsZero() {
		if err := token.Set(jwt.ExpirationKey, expires); err != nil {
			return nil, setKeyError(jwt.ExpirationKey, err)
		}
	}
	if err := token.Set(uri.ScopeKey, tokenReq.scopes); err != nil {
		return nil, setKeyError(uri.ScopeKey, err)
	}
	if err := token.Set(uri.ClientIDKey, clientCfg.ID); err != nil {
		return nil, setKeyError(uri.ClientIDKey, err)
	}
	if err := token.Set(jwt.IssuerKey, tokenReq.host); err != nil {
		return nil, setKeyError(jwt.IssuerKey, err)
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

func getSubject(clientCfg *Client, tokenReq tokenRequest) string {
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
	if tokenReq.TokenName != "" && tokenReq.ownerClaim != "name" {
		return token.Set("name", tokenReq.TokenName)
	}
	return nil
}

func setOriginTokenClaims(token jwt.Token, tokenReq tokenRequest) error {
	if len(tokenReq.originalTokenClaims) > 0 {
		return token.Set(uri.OriginalTokenClaims, tokenReq.originalTokenClaims)
	}
	return nil
}

func makeJWTPayload(key interface{}, jwkKey jwk.Key, data []byte) ([]byte, error) {
	hdr := jws.NewHeaders()
	if err := hdr.Set(jws.TypeKey, `JWT`); err != nil {
		return nil, setKeyError(jws.TypeKey, err)
	}
	if err := hdr.Set(jws.KeyIDKey, jwkKey.KeyID()); err != nil {
		return nil, setKeyError(jws.KeyIDKey, err)
	}

	payload, err := jws.Sign(data, jws.WithKey(jwkKey.Algorithm(), key, jws.WithProtectedHeaders(hdr)))
	if err != nil {
		return nil, fmt.Errorf("failed to create UserToken: %w", err)
	}
	return payload, nil
}

func generateAccessToken(clientCfg *Client, tokenReq tokenRequest, key interface{}, jwkKey jwk.Key) (string, time.Time, error) {
	now := time.Now()
	var expires time.Time
	if clientCfg.AccessTokenLifetime != 0 {
		expires = now.Add(clientCfg.AccessTokenLifetime)
	}
	token, err := makeAccessToken(clientCfg, tokenReq, now, expires)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to make token: %w", err)
	}

	buf, err := json.Encode(token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to encode token: %w", err)
	}

	payload, err := makeJWTPayload(key, jwkKey, buf)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to make payload: %w", err)
	}
	return string(payload), expires, nil
}

type tokenRequest struct {
	ClientID            string    `json:"client_id"`
	Secret              string    `json:"client_secret"`
	Audience            string    `json:"audience"`
	GrantType           GrantType `json:"grant_type"`
	ClientAssertionType string    `json:"client_assertion_type"`
	ClientAssertion     string    `json:"client_assertion"`
	TokenName           string    `json:"token_name"`

	deviceID            string          `json:"-"`
	owner               string          `json:"-"`
	subject             string          `json:"-"`
	host                string          `json:"-"`
	scopes              string          `json:"-"`
	ownerClaim          string          `json:"-"`
	deviceIDClaim       string          `json:"-"`
	tokenType           AccessTokenType `json:"-"`
	originalTokenClaims goJwt.MapClaims `json:"-"`
}

func (requestHandler *RequestHandler) getDomain() string {
	return "https://" + requestHandler.config.OAuthSigner.Domain
}

func (requestHandler *RequestHandler) postToken(w http.ResponseWriter, r *http.Request) {
	tokenReq := tokenRequest{
		host:      requestHandler.getDomain(),
		tokenType: AccessTokenType_JWT,
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		err := r.ParseForm()
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		tokenReq.GrantType = GrantType(r.PostFormValue(uri.GrantTypeKey))
		tokenReq.ClientID = r.PostFormValue(uri.ClientIDKey)
		tokenReq.Audience = r.PostFormValue(uri.AudienceKey)
		tokenReq.Secret = r.PostFormValue(uri.ClientSecretKey)
		tokenReq.ClientAssertionType = r.PostFormValue(uri.ClientAssertionTypeKey)
		tokenReq.ClientAssertion = r.PostFormValue(uri.ClientAssertionKey)
		tokenReq.TokenName = r.PostFormValue(uri.TokenName)
	} else {
		err := json.ReadFrom(r.Body, &tokenReq)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}
	clientID, secret, ok := r.BasicAuth()
	if ok {
		tokenReq.ClientID = clientID
		tokenReq.Secret = secret
	}
	requestHandler.processResponse(r.Context(), w, tokenReq)
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

func (requestHandler *RequestHandler) validateTokenRequest(ctx context.Context, clientCfg *Client, tokenReq *tokenRequest) error {
	if err := validateGrantType(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateClient(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateClientAssertionType(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := requestHandler.validateClientAssertion(ctx, tokenReq); err != nil {
		return err
	}
	if err := validateAudience(clientCfg, tokenReq); err != nil {
		return err
	}

	return nil
}

func validateClient(clientCfg *Client, tokenReq *tokenRequest) error {
	if clientCfg == nil {
		return fmt.Errorf("client(%v) not found", tokenReq.ClientID)
	}
	if clientCfg.secret != "" && !clientCfg.JWTPrivateKey.Enabled && clientCfg.secret != tokenReq.Secret {
		return errors.New("invalid client secret")
	}
	return nil
}

func validateGrantType(clientCfg *Client, tokenReq *tokenRequest) error {
	// clientCfg.AllowedGrantTypes is always non-empty
	if !sliceContains(clientCfg.AllowedGrantTypes, []GrantType{tokenReq.GrantType}) {
		return fmt.Errorf("invalid grant type(%v)", tokenReq.GrantType)
	}
	return nil
}

func validateAudience(clientCfg *Client, tokenReq *tokenRequest) error {
	var audiences []string
	if tokenReq.Audience != "" {
		audiences = []string{tokenReq.Audience}
	}
	if !sliceContains(clientCfg.AllowedAudiences, audiences) {
		return fmt.Errorf("invalid audience(%v)", tokenReq.Audience)
	}
	return nil
}

func validateClientAssertionType(clientCfg *Client, tokenReq *tokenRequest) error {
	if tokenReq.ClientAssertionType != "" && clientCfg.JWTPrivateKey.Enabled && tokenReq.ClientAssertionType != uri.ClientAssertionTypeJWT {
		return errors.New("invalid client assertion type")
	}
	return nil
}

func (requestHandler *RequestHandler) validateClientAssertion(ctx context.Context, tokenReq *tokenRequest) error {
	if tokenReq.ClientAssertionType == "" {
		return nil
	}
	v, ok := requestHandler.privateKeyJWTValidators[tokenReq.ClientID]
	if !ok {
		return errors.New("invalid client assertion")
	}
	token, err := v.GetParser().ParseWithContext(ctx, tokenReq.ClientAssertion)
	if err != nil {
		return fmt.Errorf("invalid client assertion: %w", err)
	}
	tokenReq.originalTokenClaims = token
	claims := pkgJwt.Claims(token)
	owner, err := claims.GetOwner(requestHandler.config.OAuthSigner.OwnerClaim)
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim owner: %w", err)
	}
	tokenReq.owner = owner
	sub, err := claims.GetSubject()
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim sub: %w", err)
	}
	tokenReq.subject = sub
	if requestHandler.config.OAuthSigner.DeviceIDClaim == "" {
		return nil
	}
	deviceID, err := claims.GetDeviceID(requestHandler.config.OAuthSigner.DeviceIDClaim)
	if err == nil {
		tokenReq.deviceID = deviceID
	}
	return nil
}

func (requestHandler *RequestHandler) processResponse(ctx context.Context, w http.ResponseWriter, tokenReq tokenRequest) {
	clientCfg := requestHandler.config.OAuthSigner.Clients.Find(tokenReq.ClientID)
	if clientCfg == nil {
		requestHandler.logger.Errorf("client(%v) not found - sending unauthorized", tokenReq.ClientID)
		writeError(w, errors.New("invalid client"), http.StatusUnauthorized)
		return
	}
	if err := requestHandler.validateTokenRequest(ctx, clientCfg, &tokenReq); err != nil {
		requestHandler.logger.Errorf("failed to validate token request - sending unauthorized: %w", err)
		writeError(w, errors.New("invalid client"), http.StatusUnauthorized)
		return
	}

	tokenReq.scopes = strings.Join(clientCfg.AllowedScopes, " ")
	tokenReq.deviceIDClaim = requestHandler.config.OAuthSigner.DeviceIDClaim
	tokenReq.ownerClaim = requestHandler.config.OAuthSigner.OwnerClaim

	accessToken, accessTokenExpires, err := generateAccessToken(
		clientCfg,
		tokenReq,
		requestHandler.accessTokenKey,
		requestHandler.accessTokenJwkKey)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		uri.AccessTokenKey: accessToken,
		uri.ScopeKey:       tokenReq.scopes,
		"token_type":       "Bearer",
	}
	if !accessTokenExpires.IsZero() {
		resp["expires_in"] = int64(time.Until(accessTokenExpires).Seconds())
	}

	if err = jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
		return
	}
}
