package service

import (
	cache "github.com/patrickmn/go-cache"
	"github.com/valyala/fasthttp"
)

// HandleAccessToken requests the access token for the user
func (s *Service) HandleAccessToken(ctx *fasthttp.RequestCtx) {
	if p, ok := s.deviceProvider.(interface {
		HandleAccessToken(ctx *fasthttp.RequestCtx)
	}); ok {
		p.HandleAccessToken(ctx)
		return
	}
	t, err := generateRandomString(32)
	if err != nil {
		setErrorResponse(&ctx.Response, fasthttp.StatusInternalServerError, "Random generator failed")
		return
	}
	if err := s.csrfTokens.Add(t, nil, cache.DefaultExpiration); err != nil {
		setErrorResponse(&ctx.Response, fasthttp.StatusInternalServerError, "Key collision")
		return
	}

	url := s.sdkProvider.AuthCodeURL(t)
	ctx.Redirect(url, fasthttp.StatusTemporaryRedirect)
}
