package security

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

const (
	jwksUri = "/auth/.well-known/openid-configuration/jwksX"
)

var (
	jwkRsaKey *rsa.PrivateKey
	jwkKey    jwk.Key
)

func init() {
	var err error
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	jwkRsaKey = rsaKey

	key, err := jwk.FromRaw(rsaKey.PublicKey)
	if err != nil {
		panic(err)
	}
	if err := jwk.AssignKeyID(key); err != nil {
		panic(err)
	}
	if err := key.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		panic(err)
	}
	jwkKey = key
}

type JWKServer struct {
	URI string
	*httptest.Server
}

func (j *JWKServer) URL() string {
	return j.Server.URL + j.URI
}

func CreateJwksToken(t *testing.T, claims jwt.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = jwkKey.KeyID()

	tokenString, err := token.SignedString(jwkRsaKey)
	require.NoError(t, err)

	return tokenString
}

func NewTestJwks(t *testing.T) JWKServer {
	jwks, err := json.Marshal(jwkKey)
	require.NoError(t, err)

	mux := http.NewServeMux()
	mux.HandleFunc(jwksUri, func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(jwks); err != nil {
			log.Debugf("failed to write jwks: %v", err)
		}
	})
	return JWKServer{jwksUri, httptest.NewServer(mux)}
}
