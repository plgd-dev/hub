package service

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
	"github.com/plgd-dev/hub/v2/test/oauth-server/uri"
)

func (requestHandler *RequestHandler) getOpenIDConfiguration(w http.ResponseWriter, _ *http.Request) {
	v := openid.Config{
		Issuer:             requestHandler.getDomain() + "/",
		AuthURL:            requestHandler.getDomain() + uri.Authorize,
		TokenURL:           requestHandler.getDomain() + uri.Token,
		UserInfoURL:        requestHandler.getDomain() + uri.UserInfo,
		JWKSURL:            requestHandler.getDomain() + uri.JWKs,
		EndSessionEndpoint: requestHandler.getDomain() + uri.LogOut,
	}

	if err := jsonResponseWriter(w, v); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
