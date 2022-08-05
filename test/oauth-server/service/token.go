package service

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
	"github.com/plgd-dev/kit/v2/codec/json"
)

const (
	DeviceUserID = "1"
	DefaultScope = "openid profile email offline_access r:* w:*"
)

const (
	TokenScopeKey    = "scope"
	TokenNicknameKey = "nickname"
	TokenNameKey     = "name"
	TokenPictureKey  = "picture"
)

func makeAccessToken(clientID, host, deviceID, scopes string, issuedAt, expires time.Time) (jwt.Token, error) {
	token := jwt.New()

	if err := token.Set(jwt.SubjectKey, DeviceUserID); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.SubjectKey, err)
	}
	if err := token.Set(jwt.AudienceKey, host+"/"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.AudienceKey, err)
	}
	if err := token.Set(jwt.IssuedAtKey, issuedAt); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.IssuedAtKey, err)
	}
	if !expires.IsZero() {
		if err := token.Set(jwt.ExpirationKey, expires); err != nil {
			return nil, fmt.Errorf("failed to set %v: %w", jwt.ExpirationKey, err)
		}
	}
	if err := token.Set(TokenScopeKey, scopes); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", TokenScopeKey, err)
	}
	if err := token.Set(uri.ClientIDKey, clientID); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", uri.ClientIDKey, err)
	}
	if err := token.Set(jwt.IssuerKey, host+"/"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.IssuerKey, err)
	}
	if deviceID != "" {
		if err := token.Set(uri.DeviceIDClaimKey, deviceID); err != nil {
			return nil, fmt.Errorf("failed to set %v('%v'): %w", uri.DeviceIDClaimKey, deviceID, err)
		}
	}
	// mock oauth server always set DeviceUserID, because it supports only one user
	if err := token.Set(uri.OwnerClaimKey, DeviceUserID); err != nil {
		return nil, fmt.Errorf("failed to set %v('%v'): %w", uri.OwnerClaimKey, DeviceUserID, err)
	}

	return token, nil
}

func makeJWTPayload(key interface{}, jwkKey jwk.Key, data []byte) ([]byte, error) {
	hdr := jws.NewHeaders()
	if err := hdr.Set(jws.AlgorithmKey, jwkKey.Algorithm()); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jws.AlgorithmKey, err)
	}
	if err := hdr.Set(jws.TypeKey, `JWT`); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jws.TypeKey, err)
	}
	if err := hdr.Set(jws.KeyIDKey, jwkKey.KeyID()); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jws.KeyIDKey, err)
	}
	payload, err := jws.Sign(data, jwa.SignatureAlgorithm(jwkKey.Algorithm()), key, jws.WithHeaders(hdr))
	if err != nil {
		return nil, fmt.Errorf("failed to create UserToken: %w", err)
	}
	return payload, nil
}

func generateAccessToken(clientID string, lifeTime time.Duration, host, deviceID, scopes string, key interface{}, jwkKey jwk.Key) (string, time.Time, error) {
	now := time.Now()
	var expires time.Time
	if lifeTime != 0 {
		expires = now.Add(lifeTime)
	}
	token, err := makeAccessToken(clientID, host, deviceID, scopes, now, expires)
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

func makeIDToken(clientID string, host, nonce string, issuedAt, expires time.Time) (jwt.Token, error) {
	token := jwt.New()

	if err := token.Set(jwt.SubjectKey, DeviceUserID); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.SubjectKey, err)
	}
	if err := token.Set(jwt.AudienceKey, clientID); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.AudienceKey, err)
	}
	if err := token.Set(jwt.IssuedAtKey, issuedAt); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.IssuedAtKey, err)
	}
	if expires.IsZero() {
		// itToken for UI must contains exp
		expires = time.Now().Add(time.Hour * 24 * 365 * 10)
	}
	if err := token.Set(jwt.ExpirationKey, expires); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.ExpirationKey, err)
	}
	if err := token.Set(jwt.IssuerKey, host+"/"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", jwt.IssuerKey, err)
	}
	if err := token.Set(uri.NonceKey, nonce); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", uri.NonceKey, err)
	}
	if err := token.Set(TokenNicknameKey, "test"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", TokenNicknameKey, err)
	}
	if err := token.Set(TokenNameKey, "test@test.com"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", TokenNameKey, err)
	}
	if err := token.Set(TokenPictureKey, "https://s.gravatar.com/avatar/319673928161fae8216e9a2225cff4b6?s=480&r=pg&d=https%3A%2F%2Fcdn.auth0.com%2Favatars%2Fte.png"); err != nil {
		return nil, fmt.Errorf("failed to set %v: %w", TokenPictureKey, err)
	}

	return token, nil
}

func generateIDToken(clientID string, lifeTime time.Duration, host, nonce string, key *rsa.PrivateKey, jwkKey jwk.Key) (string, error) {
	if nonce == "" {
		return "", nil
	}
	now := time.Now()
	var expires time.Time
	if lifeTime > 0 {
		expires = now.Add(lifeTime)
	}
	token, err := makeIDToken(clientID, host, nonce, now, expires)
	if err != nil {
		return "", fmt.Errorf("failed to make token: %w", err)
	}

	buf, err := json.Encode(token)
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %w", err)
	}

	payload, err := makeJWTPayload(key, jwkKey, buf)
	if err != nil {
		return "", fmt.Errorf("failed to make payload: %w", err)
	}

	return string(payload), nil
}

func generateRefreshToken(clientCfg *Client) (string, error) {
	if clientCfg.RefreshTokenRestrictionLifetime == 0 {
		return "refresh-token", nil
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (requestHandler *RequestHandler) tokenOptions(w http.ResponseWriter, r *http.Request) {
	if err := jsonResponseWriter(w, r); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

type tokenRequest struct {
	ClientID     string `json:"client_id"`
	CodeVerifier string `json:"code_verifier"`
	GrantType    string `json:"grant_type"`
	RedirectURI  string `json:"redirect_uri"`
	//	AuthorizationCode string `json:"authorization_code"`
	Code         string `json:"code"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Audience     string `json:"audience"`
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"https://plgd.dev/deviceId"`
	// mock-oauth-server always put owner claim to access token
	Owner string `json:"https://plgd.dev/owner"`

	host      string
	tokenType AccessTokenType
}

// used by acquire service token
func (requestHandler *RequestHandler) getToken(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get(uri.ClientIDKey)
	audience := r.URL.Query().Get(uri.AudienceKey)
	var ok bool
	if clientID == "" {
		clientID, _, ok = r.BasicAuth()
		if !ok {
			writeError(w, fmt.Errorf("authorization header is not set"), http.StatusBadRequest)
			return
		}
	}
	requestHandler.processResponse(w, tokenRequest{
		ClientID:  clientID,
		GrantType: string(AllowedGrantType_CLIENT_CREDENTIALS),
		Audience:  audience,

		host:      r.Host,
		tokenType: AccessTokenType_JWT,
	})
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
		tokenReq.GrantType = r.PostFormValue(uri.GrantTypeKey)
		tokenReq.ClientID = r.PostFormValue(uri.ClientIDKey)
		tokenReq.RedirectURI = r.PostFormValue(uri.RedirectURIKey)
		tokenReq.Code = r.PostFormValue(uri.CodeKey)
		tokenReq.Username = r.PostFormValue(uri.UsernameKey)
		tokenReq.Password = r.PostFormValue(uri.PasswordKey)
		tokenReq.Audience = r.PostFormValue(uri.AudienceKey)
		tokenReq.RefreshToken = r.PostFormValue(uri.RefreshTokenKey)
		tokenReq.Owner = r.PostFormValue(uri.OwnerClaimKey)
		tokenReq.DeviceID = r.PostFormValue(uri.DeviceIDClaimKey)
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
	requestHandler.processResponse(w, tokenReq)
}

func (requestHandler *RequestHandler) getAuthorizedSession(tokenReq tokenRequest) (authorizedSession, bool) {
	v := requestHandler.authSession.Load(tokenReq.Code)
	requestHandler.authSession.Delete(tokenReq.Code)
	if v != nil {
		return v.Data(), true
	}
	return authorizedSession{}, false
}

func (requestHandler *RequestHandler) validateTokenRequest(clientCfg *Client, tokenReq tokenRequest) error {
	if clientCfg == nil {
		return fmt.Errorf("client(%v) not found", tokenReq.ClientID)
	}
	if clientCfg.ClientSecret != "" && clientCfg.ClientSecret != tokenReq.Password {
		return fmt.Errorf("invalid client secret")
	}
	if clientCfg.RequiredRedirectURI != "" && clientCfg.RequiredRedirectURI != tokenReq.RedirectURI {
		return fmt.Errorf("invalid redirect uri(%v)", tokenReq.RedirectURI)
	}
	return nil
}

func (requestHandler *RequestHandler) validateRestrictions(clientCfg *Client, tokenReq tokenRequest) error {
	if clientCfg.CodeRestrictionLifetime != 0 && tokenReq.GrantType == string(AllowedGrantType_AUTHORIZATION_CODE) {
		v := requestHandler.authRestriction.Load(tokenReq.Code)
		if v != nil {
			return fmt.Errorf("auth code(%v) reused", tokenReq.Code)
		}
		requestHandler.authRestriction.LoadOrStore(tokenReq.Code, cache.NewElement(struct{}{}, time.Now().Add(clientCfg.CodeRestrictionLifetime), nil))
	}

	if tokenReq.GrantType == string(AllowedGrantType_REFRESH_TOKEN) {
		if clientCfg.RefreshTokenRestrictionLifetime != 0 {
			v := requestHandler.refreshRestriction.Load(tokenReq.RefreshToken)
			if v != nil {
				return fmt.Errorf("refresh token(%v) reused", tokenReq.RefreshToken)
			}
			requestHandler.refreshRestriction.LoadOrStore(tokenReq.RefreshToken, cache.NewElement(struct{}{}, time.Now().Add(clientCfg.RefreshTokenRestrictionLifetime), nil))
			return nil
		}
		if tokenReq.RefreshToken != "refresh-token" {
			return fmt.Errorf("invalid refresh token(%v)", tokenReq.RefreshToken)
		}
	}
	return nil
}

func (requestHandler *RequestHandler) processResponse(w http.ResponseWriter, tokenReq tokenRequest) {
	clientCfg := requestHandler.config.OAuthSigner.Clients.Find(tokenReq.ClientID)
	if err := requestHandler.validateTokenRequest(clientCfg, tokenReq); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if err := requestHandler.validateRestrictions(clientCfg, tokenReq); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	refreshToken, err := generateRefreshToken(clientCfg)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	var authSession authorizedSession
	if tokenReq.GrantType == string(AllowedGrantType_AUTHORIZATION_CODE) {
		var found bool
		authSession, found = requestHandler.getAuthorizedSession(tokenReq)
		if !found && clientCfg.RequireIssuedAuthorizationCode {
			writeError(w, fmt.Errorf("invalid authorization code(%v)", tokenReq.Code), http.StatusBadRequest)
			return
		}
	}

	if authSession.audience != "" && tokenReq.Audience == "" {
		tokenReq.Audience = authSession.audience
		tokenReq.tokenType = AccessTokenType_JWT
	}

	if authSession.deviceID != "" {
		tokenReq.DeviceID = authSession.deviceID
	}

	idToken, err := generateIDToken(tokenReq.ClientID, clientCfg.AccessTokenLifetime, tokenReq.host, authSession.nonce, requestHandler.idTokenKey, requestHandler.idTokenJwkKey)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	scopes := authSession.scope
	if tokenReq.GrantType == string(AllowedGrantType_REFRESH_TOKEN) {
		scopes = DefaultScope
	}

	accessToken, accessTokenExpires, err := generateAccessToken(clientCfg.ID, clientCfg.AccessTokenLifetime, tokenReq.host, tokenReq.DeviceID, scopes, requestHandler.accessTokenKey, requestHandler.accessTokenJwkKey)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"access_token":  accessToken,
		"id_token":      idToken,
		"scope":         scopes,
		"token_type":    "Bearer",
		"refresh_token": refreshToken,
	}
	if !accessTokenExpires.IsZero() {
		resp["expires_in"] = int64(time.Until(accessTokenExpires).Seconds())
	}

	if err = jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
		return
	}
}
