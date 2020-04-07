package service

import (
	"crypto/rand"
	"encoding/base64"

	cache "github.com/patrickmn/go-cache"
	"github.com/valyala/fasthttp"
)

// HandleAuthorizationCode requests the authorization code for the device on behalf of the user
func (s *Service) HandleAuthorizationCode(ctx *fasthttp.RequestCtx) {
	if p, ok := s.deviceProvider.(interface {
		HandleAuthorizationCode(ctx *fasthttp.RequestCtx)
	}); ok {
		p.HandleAuthorizationCode(ctx)
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

	url := s.deviceProvider.AuthCodeURL(t)
	ctx.Redirect(url, fasthttp.StatusTemporaryRedirect)
}

func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
