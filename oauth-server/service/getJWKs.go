package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"net/http"

	"github.com/go-ocf/kit/codec/json"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
)

var jwkPrivateKey *ecdsa.PrivateKey
var jwkKeyID = `QkY4MzFGMTdFMzMyN0NGQjEyOUFFMzE5Q0ZEMUYzQUQxNkNENTlEMg`
var jwkKey jwk.Key

func init() {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate key: %s", err)
	}
	jwkPrivateKey = privateKey
	key, err := jwk.New(&jwkPrivateKey.PublicKey)
	if err != nil {
		log.Fatalf("failed to create JWK: %s", err)
	}
	key.Set(jwk.KeyIDKey, key.KeyID())
	key.Set(jwk.AlgorithmKey, jwa.ES256.String())
	jwkKey = key
}

func (requestHandler *RequestHandler) getJWKs(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"keys": []jwk.Key{
			jwkKey,
		},
	}
	data, err := json.Encode(resp)
	if err != nil {
		writeError(w, err)
		return
	}

	jsonResponseWriter(w, data)
}
