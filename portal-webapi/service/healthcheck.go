package service

import "github.com/valyala/fasthttp"

func (r *RequestHandler) healthcheck(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
}
