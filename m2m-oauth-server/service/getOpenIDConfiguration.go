package service

import (
	"net/http"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/uri"
	"github.com/plgd-dev/hub/v2/pkg/log"
	"github.com/plgd-dev/hub/v2/pkg/security/openid"
)

func (requestHandler *RequestHandler) getOpenIDConfiguration(w http.ResponseWriter, _ *http.Request) {
	v := openid.Config{
		Issuer:   requestHandler.getDomain() + "/",
		TokenURL: requestHandler.getDomain() + uri.Token,
		JWKSURL:  requestHandler.getDomain() + uri.JWKs,
	}

	if err := jsonResponseWriter(w, v); err != nil {
		log.Errorf("failed to write response: %v", err)
	}
}
