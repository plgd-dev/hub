package service

import "github.com/valyala/fasthttp"

func setErrorResponse(r *fasthttp.Response, code int, body string) {
	r.Header.SetContentType("text/plain; charset=utf-8")
	r.SetStatusCode(code)
	r.SetBodyString(body)
}
