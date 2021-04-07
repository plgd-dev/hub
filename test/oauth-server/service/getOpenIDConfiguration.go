package service

import (
	"net/http"

	"github.com/plgd-dev/cloud/pkg/security/jwt/validator"
	"github.com/plgd-dev/cloud/test/oauth-server/uri"
)

func (requestHandler *RequestHandler) getOpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	v := validator.OpenIDConfiguration{
		Issuer:      requestHandler.getDomain() + "/",
		AuthURL:     requestHandler.getDomain() + uri.Authorize,
		TokenURL:    requestHandler.getDomain() + uri.Token,
		UserInfoURL: requestHandler.getDomain() + uri.UserInfo,
		JWKSURL:     requestHandler.getDomain() + uri.JWKs,
	}

	jsonResponseWriter(w, v)
}
