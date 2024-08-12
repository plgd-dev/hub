package http

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
)

func GetOpenIDConfiguration(domain string) openid.Config {
	return openid.Config{
		Issuer:             domain + uri.Base,
		TokenURL:           domain + uri.Token,
		JWKSURL:            domain + uri.JWKs,
		PlgdTokensEndpoint: domain + uri.Tokens,
	}
}

func (requestHandler *RequestHandler) getOpenIDConfiguration(w http.ResponseWriter, _ *http.Request) {
	v := GetOpenIDConfiguration(requestHandler.m2mOAuthServiceServer.GetDomain())

	if err := jsonResponseWriter(w, v); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
