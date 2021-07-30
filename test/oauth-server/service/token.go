package service

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/plgd-dev/cloud/pkg/log"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
	"github.com/plgd-dev/kit/codec/json"
)

var DeviceUserID = "1"

const (
	TokenScopeKey    = "scope"
	TokenNicknameKey = "nickname"
	TokenNameKey     = "name"
	TokenPictureKey  = "picture"
)

// Token provides access tokens and their attributes.
type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	UserID       string
}

func generateAccessToken(clientID string, lifeTime time.Duration, host string, key interface{}, jwkKey jwk.Key) (string, time.Time, error) {
	token := jwt.New()
	now := time.Now()
	expires := now.Add(lifeTime)

	if err := token.Set(jwt.SubjectKey, DeviceUserID); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jwt.SubjectKey, err)
	}
	if err := token.Set(jwt.AudienceKey, host+"/"); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jwt.AudienceKey, err)
	}
	if err := token.Set(jwt.IssuedAtKey, now); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jwt.IssuedAtKey, err)
	}
	if err := token.Set(jwt.ExpirationKey, expires); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jwt.ExpirationKey, err)
	}
	if err := token.Set(TokenScopeKey, []string{"openid", "r:deviceinformation:*", "r:resources:*", "w:resources:*", "w:subscriptions:*"}); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", TokenScopeKey, err)
	}
	if err := token.Set(uri.ClientIDKey, clientID); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", uri.ClientIDKey, err)
	}
	if err := token.Set(jwt.IssuerKey, host+"/"); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jwt.IssuerKey, err)
	}
	buf, err := json.Encode(token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to encode token: %s", err)
	}

	hdr := jws.NewHeaders()
	if err = hdr.Set(jws.AlgorithmKey, jwkKey.Algorithm()); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jws.AlgorithmKey, err)
	}
	if err = hdr.Set(jws.TypeKey, `JWT`); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jws.TypeKey, err)
	}
	kid := jwkKey.KeyID()
	if err = hdr.Set(jws.KeyIDKey, kid); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to set %v: %w", jws.KeyIDKey, err)
	}
	payload, err := jws.Sign(buf, jwa.SignatureAlgorithm(jwkKey.Algorithm()), key, jws.WithHeaders(hdr))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create UserToken: %s", err)
	}
	return string(payload), expires, nil
}

func generateIDToken(clientID string, lifeTime time.Duration, host, nonce string, key *rsa.PrivateKey, jwkKey jwk.Key) (string, error) {
	token := jwt.New()
	now := time.Now()
	expires := now.Add(lifeTime)

	if err := token.Set(jwt.SubjectKey, DeviceUserID); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jwt.SubjectKey, err)
	}
	if err := token.Set(jwt.AudienceKey, clientID); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jwt.AudienceKey, err)
	}
	if err := token.Set(jwt.IssuedAtKey, now); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jwt.IssuedAtKey, err)
	}
	if err := token.Set(jwt.ExpirationKey, expires); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jwt.ExpirationKey, err)
	}
	if err := token.Set(jwt.IssuerKey, host+"/"); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jwt.IssuerKey, err)
	}
	if err := token.Set(uri.NonceKey, nonce); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", uri.NonceKey, err)
	}
	if err := token.Set(TokenNicknameKey, "test"); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", TokenNicknameKey, err)
	}
	if err := token.Set(TokenNameKey, "test@test.com"); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", TokenNameKey, err)
	}
	if err := token.Set(TokenPictureKey, "https://s.gravatar.com/avatar/319673928161fae8216e9a2225cff4b6?s=480&r=pg&d=https%3A%2F%2Fcdn.auth0.com%2Favatars%2Fte.png"); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", TokenPictureKey, err)
	}
	//,\"updated_at\":\"2021-02-24T08:13:30.677Z\",\"email\":\"dnaik@infinera.com\",\"email_verified\":true,
	buf, err := json.Encode(token)
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %s", err)
	}

	hdr := jws.NewHeaders()
	if err = hdr.Set(jws.AlgorithmKey, jwkKey.Algorithm()); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jws.AlgorithmKey, err)
	}
	if err = hdr.Set(jws.TypeKey, `JWT`); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jws.TypeKey, err)
	}
	if err = hdr.Set(jws.KeyIDKey, jwkKey.KeyID()); err != nil {
		return "", fmt.Errorf("failed to set %v: %w", jws.KeyIDKey, err)
	}
	payload, err := jws.Sign(buf, jwa.SignatureAlgorithm(jwkKey.Algorithm()), key, jws.WithHeaders(hdr))
	if err != nil {
		return "", fmt.Errorf("failed to create UserToken: %s", err)
	}
	return string(payload), nil
}

func (requestHandler *RequestHandler) tokenOptions(w http.ResponseWriter, r *http.Request) {
	if err := jsonResponseWriter(w, r); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}

type tokenRequest struct {
	// RedirectURI  string `json:"redirect_uri"`
	ClientID     string `json:"client_id"`
	CodeVerifier string `json:"code_verifier"`
	GrantType    string `json:"grant_type"`
	//	AuthorizationCode string `json:"authorization_code"`
	Code     string `json:"code"`
	Username string `json:"username"`
	Password string `json:"password"`
	Audience string `json:"audience"`

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
		tokenType: AccessTokenType_REFERENCE,
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		err := r.ParseForm()
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
		tokenReq.GrantType = r.PostFormValue(uri.GrantTypeKey)
		tokenReq.ClientID = r.PostFormValue(uri.ClientIDKey)
		tokenReq.Code = r.PostFormValue(uri.CodeKey)
		tokenReq.Username = r.PostFormValue(uri.UsernameKey)
		tokenReq.Password = r.PostFormValue(uri.PasswordKey)
		tokenReq.Audience = r.PostFormValue(uri.AudienceKey)
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

	if tokenReq.Audience != "" {
		tokenReq.tokenType = AccessTokenType_JWT
	}
	requestHandler.processResponse(w, tokenReq)
}

func (requestHandler *RequestHandler) processResponse(w http.ResponseWriter, tokenReq tokenRequest) {
	clientCfg := clients.Find(tokenReq.ClientID)
	if clientCfg == nil {
		writeError(w, fmt.Errorf("client(%v) not found", tokenReq.ClientID), http.StatusBadRequest)
		return
	}
	var authSession authorizedSession
	authSessionI, ok := requestHandler.cache.Get(tokenReq.Code)
	requestHandler.cache.Delete(tokenReq.Code)
	if ok {
		authSession = authSessionI.(authorizedSession)
		if authSession.audience != "" && tokenReq.Audience == "" {
			tokenReq.Audience = authSession.audience
			tokenReq.tokenType = AccessTokenType_JWT
		}
	}
	var idToken string
	var accessToken string
	var accessTokenExpires time.Time
	var err error
	if authSession.nonce != "" {
		idToken, err = generateIDToken(tokenReq.ClientID, clientCfg.AccessTokenLifetime, tokenReq.host, authSession.nonce, requestHandler.idTokenKey, requestHandler.idTokenJwkKey)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
	}
	if tokenReq.tokenType == AccessTokenType_JWT {
		accessToken, accessTokenExpires, err = generateAccessToken(clientCfg.ID, clientCfg.AccessTokenLifetime, tokenReq.host, requestHandler.accessTokenKey, requestHandler.accessTokenJwkKey)
		if err != nil {
			writeError(w, err, http.StatusInternalServerError)
			return
		}
	} else {
		accessToken = clientCfg.ID
		accessTokenExpires = time.Now().Add(clientCfg.AccessTokenLifetime)
	}
	resp := map[string]interface{}{
		"access_token":  accessToken,
		"id_token":      idToken,
		"expires_in":    int64(time.Until(accessTokenExpires).Seconds()),
		"scope":         "openid profile email",
		"token_type":    "Bearer",
		"refresh_token": "refresh-token",
	}

	if err = jsonResponseWriter(w, resp); err != nil {
		log.Errorf("failed to write response: %v", err)
		return
	}
}
