package service

import (
	"time"

	"github.com/go-ocf/kit/codec/json"
	"github.com/valyala/fasthttp"
)

func (p *TestProvider) HandleAccessToken(ctx *fasthttp.RequestCtx) {
	clientID := string(ctx.QueryArgs().Peek("ClientId"))
	var isService bool
	if clientID == "service" {
		isService = true
	}
	token, err := generateToken(isService)
	if err != nil {
		setErrorResponse(&ctx.Response, fasthttp.StatusInternalServerError, err.Error())
		return
	}
	resp := map[string]interface{}{
		"access_token": token.AccessToken,
		"expires_in":   int64(token.Expiry.Sub(time.Now()).Seconds()),
		"scope":        "openid",
		"token_type":   "Bearer",
	}
	data, err := json.Encode(resp)
	if err != nil {
		setErrorResponse(&ctx.Response, fasthttp.StatusInternalServerError, err.Error())
		return
	}
	r := &ctx.Response
	r.Header.SetContentType("application/json")
	r.SetStatusCode(fasthttp.StatusOK)
	r.SetBodyString(string(data))
}
