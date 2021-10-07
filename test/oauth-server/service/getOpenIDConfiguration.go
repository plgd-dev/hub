package service

import (
	"net/http"

	"github.com/plgd-dev/hub/pkg/log"
	"github.com/plgd-dev/hub/pkg/security/openid"
	"github.com/plgd-dev/hub/test/oauth-server/uri"
)

func (requestHandler *RequestHandler) getOpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	v := openid.Config{
		Issuer:      requestHandler.getDomain() + "/",
		AuthURL:     requestHandler.getDomain() + uri.Authorize,
		TokenURL:    requestHandler.getDomain() + uri.Token,
		UserInfoURL: requestHandler.getDomain() + uri.UserInfo,
		JWKSURL:     requestHandler.getDomain() + uri.JWKs,
	}

	if err := jsonResponseWriter(w, v); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
