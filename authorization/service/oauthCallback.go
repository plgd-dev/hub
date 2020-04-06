package service

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

func responseToMap(ctx *fasthttp.RequestCtx) map[string]interface{} {
	res := make(map[string]interface{})
	ctx.QueryArgs().VisitAll(func(key, val []byte) {
		res[string(key)] = string(val)
	})
	ctx.PostArgs().VisitAll(func(key, val []byte) {
		res[string(key)] = string(val)
	})
	mf, err := ctx.MultipartForm()
	if err == nil && mf.Value != nil {
		for key, val := range mf.Value {
			// only one value is stored under key: https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html
			if len(val) == 1 {
				res[key] = val[0]
			}
		}
	}
	return res
}

func (s *Service) HandleOAuthCallback(ctx *fasthttp.RequestCtx) {
	state := ctx.FormValue("state")
	_, ok := s.csrfTokens.Get(string(state))
	if !ok {
		setErrorResponse(&ctx.Response, fasthttp.StatusBadRequest, "invalid/expired OAuth state")
		return
	}
	payload, _ := json.Marshal(responseToMap(ctx))
	body := string(payload)
	setHTMLResponse(&ctx.Response, body)
}

func setHTMLResponse(r *fasthttp.Response, html string) {
	r.Header.SetContentType("text/html") // for IE - otherwise it want to download content as file
	r.SetStatusCode(fasthttp.StatusOK)
	r.SetBodyString(html)
}
