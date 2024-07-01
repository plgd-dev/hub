package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	if err := setOwnerClaim(token, tokenReq); err != nil {
		return nil, err
	}

	return token, nil
}

func getSubject(clientCfg *Client, tokenReq tokenRequest) string {
	if tokenReq.Subject != "" {
		return tokenReq.Subject
	}
	if tokenReq.Owner != "" {
		return tokenReq.Owner
	}
	return clientCfg.ID
}

func setDeviceIDClaim(token jwt.Token, tokenReq tokenRequest) error {
	if tokenReq.DeviceID != "" && tokenReq.deviceIDClaim != "" {
		return token.Set(tokenReq.deviceIDClaim, tokenReq.DeviceID)
	}
	return nil
}

func setOwnerClaim(token jwt.Token, tokenReq tokenRequest) error {
	if tokenReq.Owner != "" && tokenReq.ownerClaim != "" {
		return token.Set(tokenReq.ownerClaim, tokenReq.Owner)
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

func (requestHandler *RequestHandler) tokenOptions(w http.ResponseWriter, r *http.Request) {
	if err := jsonResponseWriter(w, r); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

type tokenRequest struct {
	ClientID            string    `json:"client_id"`
	GrantType           GrantType `json:"grant_type"`
	Username            string    `json:"username"`
	Password            string    `json:"password"`
	Audience            string    `json:"audience"`
	DeviceID            string    `json:"https://plgd.dev/deviceId"`
	Owner               string    `json:"https://plgd.dev/owner"`
	Subject             string    `json:"sub"`
	ClientAssertionType string    `json:"client_assertion_type"`
	ClientAssertion     string    `json:"client_assertion"`

	host          string
	scopes        string
	ownerClaim    string
	deviceIDClaim string
	tokenType     AccessTokenType
}

// used by acquire service token
func (requestHandler *RequestHandler) getToken(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get(uri.ClientIDKey)
	audience := r.URL.Query().Get(uri.AudienceKey)
	deviceID := r.URL.Query().Get(uri.DeviceIDKey)
	owner := r.URL.Query().Get(uri.OwnerKey)
	clientAssertionType := r.URL.Query().Get(uri.ClientAssertionTypeKey)
	clientAssertion := r.URL.Query().Get(uri.ClientAssertionKey)
	var ok bool
	if clientID == "" {
		clientID, _, ok = r.BasicAuth()
		if !ok {
			writeError(w, errors.New("authorization header is not set"), http.StatusBadRequest)
			return
		}
	}
	tr := tokenRequest{
		ClientID:            clientID,
		GrantType:           GrantTypeClientCredentials,
		Audience:            audience,
		DeviceID:            deviceID,
		Owner:               owner,
		ClientAssertionType: clientAssertionType,
		ClientAssertion:     clientAssertion,

		host:      r.Host,
		tokenType: AccessTokenType_JWT,
	}
	requestHandler.processResponse(r.Context(), w, tr)
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
		tokenReq.Username = r.PostFormValue(uri.UsernameKey)
		tokenReq.Password = r.PostFormValue(uri.PasswordKey)
		tokenReq.Audience = r.PostFormValue(uri.AudienceKey)
		tokenReq.Owner = r.PostFormValue(uri.OwnerKey)
		tokenReq.DeviceID = r.PostFormValue(uri.DeviceIDKey)
		tokenReq.Subject = r.PostFormValue(uri.SubjectKey)
		tokenReq.ClientAssertionType = r.PostFormValue(uri.ClientAssertionTypeKey)
		tokenReq.ClientAssertion = r.PostFormValue(uri.ClientAssertionKey)
	} else {
		err := json.ReadFrom(r.Body, &tokenReq)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}
	clientID, password, ok := r.BasicAuth()
	if ok {
		tokenReq.ClientID = clientID
		tokenReq.Password = password
	}
	requestHandler.processResponse(r.Context(), w, tokenReq)
}

func sliceContains[T comparable](s []T, sub []T) bool {
	if len(s) == 0 {
		return true
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
	if err := requestHandler.validateClientAssertion(ctx, clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateAudience(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateDeviceID(clientCfg, tokenReq); err != nil {
		return err
	}
	if err := validateOwner(clientCfg, tokenReq); err != nil {
		return err
	}

	return nil
}

func validateClient(clientCfg *Client, tokenReq *tokenRequest) error {
	if clientCfg == nil {
		return fmt.Errorf("client(%v) not found", tokenReq.ClientID)
	}
	if clientCfg.secret != "" && !clientCfg.PrivateKeyJWT.Enabled && clientCfg.secret != tokenReq.Password {
		return errors.New("invalid client secret")
	}
	return nil
}

func validateGrantType(clientCfg *Client, tokenReq *tokenRequest) error {
	if !sliceContains(clientCfg.AllowedGrantTypes, []GrantType{tokenReq.GrantType}) {
		return fmt.Errorf("invalid grant type(%v)", tokenReq.GrantType)
	}
	return nil
}

func validateAudience(clientCfg *Client, tokenReq *tokenRequest) error {
	if !sliceContains(clientCfg.AllowedAudiences, []string{tokenReq.Audience}) {
		return fmt.Errorf("invalid audience(%v)", tokenReq.Audience)
	}
	return nil
}

func validateDeviceID(clientCfg *Client, tokenReq *tokenRequest) error {
	if clientCfg.RequireDeviceID && tokenReq.DeviceID == "" {
		return errors.New("deviceID is required")
	}
	return nil
}

func validateOwner(clientCfg *Client, tokenReq *tokenRequest) error {
	if clientCfg.RequireOwner && tokenReq.Owner == "" {
		return errors.New("owner is required")
	}
	return nil
}

func validateClientAssertionType(clientCfg *Client, tokenReq *tokenRequest) error {
	if tokenReq.ClientAssertionType != "" && clientCfg.PrivateKeyJWT.Enabled && tokenReq.ClientAssertionType != uri.ClientAssertionTypeJWT {
		return errors.New("invalid client assertion type")
	}
	return nil
}

func (requestHandler *RequestHandler) validateClientAssertion(ctx context.Context, clientCfg *Client, tokenReq *tokenRequest) error {
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
	claims := pkgJwt.Claims(token)
	owner, err := claims.GetOwner(requestHandler.config.OAuthSigner.OwnerClaim)
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim owner: %w", err)
	}
	tokenReq.Owner = owner
	sub, err := claims.GetSubject()
	if err != nil {
		return fmt.Errorf("invalid client assertion - claim sub: %w", err)
	}
	tokenReq.Subject = sub
	deviceID, err := claims.GetDeviceID(requestHandler.config.OAuthSigner.DeviceIDClaim)
	if err == nil {
		tokenReq.DeviceID = deviceID
	}
	return nil
}

func (requestHandler *RequestHandler) processResponse(ctx context.Context, w http.ResponseWriter, tokenReq tokenRequest) {
	clientCfg := requestHandler.config.OAuthSigner.Clients.Find(tokenReq.ClientID)
	if err := requestHandler.validateTokenRequest(ctx, clientCfg, &tokenReq); err != nil {
		writeError(w, err, http.StatusBadRequest)
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
