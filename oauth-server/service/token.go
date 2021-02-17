package service

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/plgd-dev/cloud/oauth-server/uri"
	"github.com/plgd-dev/kit/codec/json"
)

var deviceUserID = "1"

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
	token.Set(jwt.SubjectKey, deviceUserID)
	token.Set(jwt.AudienceKey, clientID)
	token.Set(jwt.IssuedAtKey, now)
	token.Set(jwt.ExpirationKey, expires)
	token.Set(`scope`, []string{"openid", "r:deviceinformation:*", "r:resources:*", "w:resources:*", "w:subscriptions:*"})
	token.Set(uri.ClientIDQueryKey, clientID)
	token.Set(jwt.IssuerKey, "https://"+host+"/")
	buf, err := json.Encode(token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to encode token: %s", err)
	}

	hdr := jws.NewHeaders()
	hdr.Set(jws.AlgorithmKey, jwkKey.Algorithm())
	hdr.Set(jws.TypeKey, `JWT`)
	kid := jwkKey.KeyID()
	hdr.Set(jws.KeyIDKey, kid)
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
	token.Set(jwt.SubjectKey, deviceUserID)
	token.Set(jwt.AudienceKey, clientID)
	token.Set(jwt.IssuedAtKey, now)
	token.Set(jwt.ExpirationKey, expires)
	token.Set(jwt.IssuerKey, "https://"+host+"/")
	token.Set(uri.NonceQueryKey, nonce)
	buf, err := json.Encode(token)
	if err != nil {
		return "", fmt.Errorf("failed to encode token: %s", err)
	}

	hdr := jws.NewHeaders()
	hdr.Set(jws.AlgorithmKey, jwkKey.Algorithm())
	hdr.Set(jws.TypeKey, `JWT`)
	hdr.Set(jws.KeyIDKey, jwkKey.KeyID())
	payload, err := jws.Sign(buf, jwa.SignatureAlgorithm(jwkKey.Algorithm()), key, jws.WithHeaders(hdr))
	if err != nil {
		return "", fmt.Errorf("failed to create UserToken: %s", err)
	}
	return string(payload), nil
}

func (requestHandler *RequestHandler) tokenOptions(w http.ResponseWriter, r *http.Request) {
	jsonResponseWriter(w, r)
}

type tokenRequest struct {
	RedirectURI       string `json:"redirect_uri"`
	ClientID          string `json:"client_id"`
	CodeVerifier      string `json:"code_verifier"`
	GrantType         string `json:"grant_type"`
	AuthorizationCode string `json:"authorization_code"`
	Code              string `json:"code"`
}

func (requestHandler *RequestHandler) token(w http.ResponseWriter, r *http.Request) {
	var tokenReq tokenRequest
	clientID, _, ok := r.BasicAuth()
	if ok {
		tokenReq.ClientID = clientID
		tokenReq.GrantType = string(AllowedGrantType_CLIENT_CREDENTIALS)
	} else {
		err := json.ReadFrom(r.Body, &tokenReq)
		if err != nil {
			writeError(w, err, http.StatusBadRequest)
			return
		}
	}

	var clientCfg *Client
	var idToken string
	var err error
	if tokenReq.GrantType == string(AllowedGrantType_AUTHORIZATION_CODE) {
		authSessionI, ok := requestHandler.cache.Get(tokenReq.Code)
		if !ok {
			writeError(w, fmt.Errorf("invalid code '%v'", tokenReq.Code), http.StatusInternalServerError)
			return
		}
		requestHandler.cache.Delete(tokenReq.Code)
		authSession := authSessionI.(authorizedSession)

		if tokenReq.ClientID != authSession.cfg.ID {
			writeError(w, fmt.Errorf("invalid client_id(%v)", tokenReq.ClientID), http.StatusBadRequest)
			return
		}
		clientCfg = authSession.cfg

		if authSession.cfg.AccessTokenType == AccessTokenType_JWT {
			idToken, err = generateIDToken(authSession.cfg.ID, authSession.cfg.AccessTokenLifetime, r.Host, authSession.nonce, requestHandler.idTokenKey, requestHandler.idTokenJwkKey)
			if err != nil {
				writeError(w, err, http.StatusInternalServerError)
				return
			}
		}
	} else if tokenReq.GrantType == string(AllowedGrantType_CLIENT_CREDENTIALS) {
		clientCfg = requestHandler.config.Clients.Find(tokenReq.ClientID)
	}
	if clientCfg == nil {
		writeError(w, fmt.Errorf("invalid client_id(%v)", tokenReq.ClientID), http.StatusBadRequest)
		return
	}
	var accessToken string
	var accessTokenExpires time.Time

	if clientCfg.AccessTokenType == AccessTokenType_JWT {
		accessToken, accessTokenExpires, err = generateAccessToken(clientCfg.ID, clientCfg.AccessTokenLifetime, r.Host, requestHandler.accessTokenKey, requestHandler.accessTokenJwkKey)
	} else if clientCfg.AccessTokenType == AccessTokenType_REFERENCE {
		accessToken = clientCfg.ID
		accessTokenExpires = time.Now().Add(clientCfg.AccessTokenLifetime)
	}
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"access_token":  accessToken,
		"id_token":      idToken,
		"expires_in":    int64(accessTokenExpires.Sub(time.Now()).Seconds()),
		"scope":         "openid profile email",
		"token_type":    "Bearer",
		"refresh_token": "refresh",
	}

	jsonResponseWriter(w, resp)
}
