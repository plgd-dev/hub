package service

import (
	"github.com/valyala/fasthttp"
)

func (s *Service) Healthcheck(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
}
